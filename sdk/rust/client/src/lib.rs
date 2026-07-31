// nmid client.
// 对应 node sdk: lib/client/client.js
//
// 使用 tokio 异步运行时：
// - 连接 nmid server，split 为 read/write 两半
// - reader task 循环读取 packet，解码后按 handle 分发给 pending 请求
// - do_async 返回 Future，响应到达时 resolve

pub mod request;
pub mod response;

pub use request::*;
pub use response::*;

use std::collections::{HashMap, VecDeque};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use byteorder::{BigEndian, ByteOrder};
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::TcpStream;
use tokio::net::tcp::OwnedWriteHalf;
use tokio::sync::{oneshot, Mutex};
use tokio::time::timeout;

pub struct Client {
    addr: String,
    // write half 受锁保护，do_async 发送请求时持锁
    writer: Arc<Mutex<Option<OwnedWriteHalf>>>,
    // handle -> 等待该函数响应的请求队列（FIFO）
    pending: Arc<Mutex<HashMap<String, VecDeque<oneshot::Sender<Result<Response, ClientError>>>>>>,
    closed: Arc<AtomicBool>,
}

impl Client {
    pub fn new(addr: &str) -> Self {
        Client {
            addr: addr.to_string(),
            writer: Arc::new(Mutex::new(None)),
            pending: Arc::new(Mutex::new(HashMap::new())),
            closed: Arc::new(AtomicBool::new(false)),
        }
    }

    // 连接到 nmid server，启动 reader task
    pub async fn start(&self) -> Result<(), ClientError> {
        let stream = timeout(model::DIAL_TIME_OUT, TcpStream::connect(&self.addr))
            .await
            .map_err(|_| ClientError::InvalidData("connect timeout".into()))??;

        let (read_half, write_half) = stream.into_split();
        *self.writer.lock().await = Some(write_half);

        let pending = Arc::clone(&self.pending);
        let closed = Arc::clone(&self.closed);

        // 启动读取协程：循环读取并重组 packet，解码后分发
        tokio::spawn(async move {
            let mut reader = read_half;
            let mut buf = vec![0u8; 4096];
            let mut recv: Vec<u8> = Vec::new();

            loop {
                if closed.load(Ordering::SeqCst) {
                    break;
                }
                match reader.read(&mut buf).await {
                    Ok(0) => break, // 连接关闭
                    Ok(n) => {
                        recv.extend_from_slice(&buf[..n]);
                        // 从缓冲区重组完整 packet
                        loop {
                            if recv.len() < model::MIN_DATA_SIZE {
                                break;
                            }
                            let data_len =
                                BigEndian::read_u32(&recv[8..model::MIN_DATA_SIZE]) as usize;
                            let packet_len = model::MIN_DATA_SIZE + data_len;
                            if recv.len() < packet_len {
                                break;
                            }
                            let packet: Vec<u8> = recv.drain(..packet_len).collect();

                            // 只处理来自 server 的包
                            if get_conn_type(&packet) != model::CONN_TYPE_SERVER {
                                continue;
                            }

                            match Response::decode_pack(&packet) {
                                Ok((resp, _)) => {
                                    Self::process_response(&pending, resp).await;
                                }
                                Err(e) => {
                                    log::warn!("decode error: {}", e);
                                }
                            }
                        }
                    }
                    Err(e) => {
                        log::warn!("read error: {}", e);
                        break;
                    }
                }
            }

            // 连接断开，fail all pending
            Self::fail_all(&pending, || {
                ClientError::InvalidData("connection closed".into())
            })
            .await;
        });

        Ok(())
    }

    // 处理一个解码后的 response：错误类型 fail all，返回数据则按 handle 分发
    async fn process_response(
        pending: &Arc<Mutex<HashMap<String, VecDeque<oneshot::Sender<Result<Response, ClientError>>>>>>,
        resp: Response,
    ) {
        // 全局协议错误（PDT_ERROR/PDT_CANT_DO/PDT_RATELIMIT）：拒绝所有 pending
        if ClientError::from_data_type(resp.data_type).is_some() {
            let dt = resp.data_type;
            Self::fail_all(pending, move || ClientError::from_data_type(dt).unwrap()).await;
            return;
        }

        // 正常返回数据：按 handle 分发给队首请求
        if resp.data_type == model::PDT_S_RETURN_DATA {
            let mut p = pending.lock().await;
            if let Some(queue) = p.get_mut(&resp.handle) {
                if let Some(tx) = queue.pop_front() {
                    let _ = tx.send(Ok(resp));
                }
            }
        }
    }

    // 把所有 pending 请求都 reject
    // 用闭包生成错误，避免 ClientError 需要实现 Clone
    async fn fail_all<F>(
        pending: &Arc<Mutex<HashMap<String, VecDeque<oneshot::Sender<Result<Response, ClientError>>>>>>,
        make_err: F,
    ) where
        F: Fn() -> ClientError,
    {
        let mut p = pending.lock().await;
        for (_, queue) in p.drain() {
            for tx in queue {
                let _ = tx.send(Err(make_err()));
            }
        }
    }

    // 异步调用：发送 job 请求，返回 Future，响应到达时 resolve
    // 对应 node sdk: client.doAsync
    pub async fn do_async(
        &self,
        func_name: &str,
        params: Vec<u8>,
    ) -> Result<Response, ClientError> {
        if self.closed.load(Ordering::SeqCst) {
            return Err(ClientError::InvalidData("connection is not established".into()));
        }

        let (tx, rx) = oneshot::channel();
        {
            let mut p = self.pending.lock().await;
            p.entry(func_name.to_string())
                .or_default()
                .push_back(tx);
        }

        // 编码并发送请求
        let mut req = Request::new();
        req.content_pack(model::PDT_C_DO_JOB, func_name.to_string(), params);
        let buf = req.encode_pack();

        {
            let mut w = self.writer.lock().await;
            if let Some(stream) = w.as_mut() {
                if let Err(e) = stream.write_all(&buf).await {
                    // 写失败：连接已断，reader task 会 fail all pending
                    return Err(ClientError::Io(e));
                }
            } else {
                return Err(ClientError::InvalidData(
                    "connection is not established".into(),
                ));
            }
        }

        // 等待响应
        rx.await
            .map_err(|_| ClientError::InvalidData("request canceled".into()))?
    }

    pub async fn close(&self) {
        self.closed.store(true, Ordering::SeqCst);
        {
            let mut w = self.writer.lock().await;
            if let Some(mut s) = w.take() {
                let _ = s.shutdown().await;
            }
        }
        Self::fail_all(&self.pending, || {
            ClientError::InvalidData("client closed".into())
        })
        .await;
    }
}
