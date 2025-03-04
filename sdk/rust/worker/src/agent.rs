use super::{Worker,Request,Response,utils,ResponseError};
use std::sync::atomic::{AtomicU64, AtomicBool, Ordering};
use std::sync::Arc;
use tokio::net::TcpStream;
use tokio::sync::Mutex;
use tokio::io::{AsyncReadExt, AsyncWriteExt, ErrorKind};
use tokio::time::timeout;
use std::io;
use thiserror::Error;
use byteorder::BigEndian;

#[derive(Error, Debug)]
pub enum AgentError {
    #[error("I/O error: {0}")]
    Io(#[from] io::Error),
    #[error("Connection not initialized")]
    ConnectionUninitialized,
    #[error("NotConnected")]
    NotConnected,
    #[error("Connection lost")]
    ConnectionLost,
    #[error("Insufficient header data")]
    InsufficientHeaderData,
    #[error("Premature connection close")]
    PrematureConnectionClose,
    #[error("WriteError")]
    WriteError,
}

#[derive(Debug)]
pub struct Agent {
    net: String,
    pub addr: String,
    pub conn: Arc<Mutex<Option<TcpStream>>>,
    pub worker: Arc<Mutex<Worker>>,
    pub req: Arc<Mutex<Request>>,
    pub res: Arc<Mutex<Response>>,
    pub last_time: Arc<AtomicU64>,
    cancel_flag: Arc<AtomicBool>,
    left_data: Vec<u8>,
}

impl Agent {
    pub fn new(net: &str, addr: &str, worker: Arc<Mutex<Worker>>) -> Self {
        Agent {
            net: net.to_string(),
            addr: addr.to_string(),
            conn: Arc::new(Mutex::new(None)),
            worker,
            req: Arc::new(Mutex::new(Request::new())),
            res: Arc::new(Mutex::new(Response::new())),
            last_time: Arc::new(AtomicU64::new(0)),
            cancel_flag: Arc::new(AtomicBool::new(false)),
            left_data: vec![],
        }
    }

    fn clone_agent(&self) -> Self {
        Self {
            net: self.net.clone(),
            addr: self.addr.clone(),
            conn: self.conn.clone(),
            worker: self.worker.clone(),
            req: self.req.clone(),
            res: self.res.clone(),
            last_time: self.last_time.clone(),
            cancel_flag: self.cancel_flag.clone(),
            left_data: self.left_data.clone(),
        }
    }

    pub async fn connect(&self) -> Result<(), Box<dyn std::error::Error+Send+Sync>> {
        let stream = timeout(
            model::DIAL_TIME_OUT,
            TcpStream::connect(format!("{}:{}", self.net, self.addr)),
        )
        .await??;

        let mut guard = self.conn.lock().await;
        *guard = Some(stream);

        self.update_last_time();

        Ok(())
    }

    pub async fn read(&self) -> Result<Vec<u8>, AgentError> {
        let mut buf = Vec::with_capacity(model::MIN_DATA_SIZE * 2);
        let mut temp = utils::get_buffer(model::MIN_DATA_SIZE);
        
        let mut conn_guard = self.conn.lock().await;
        let conn = conn_guard.as_mut().ok_or(AgentError::NotConnected)?; // 明确处理未连接状态
        if let Err(e) = conn.peer_addr() {
            *conn_guard = None;
            return Err(AgentError::Io(e))
        }

        // 读取初始数据块
        let n = conn.read(&mut temp).await?;
        if n < model::MIN_DATA_SIZE {
            return Err(
                AgentError::InsufficientHeaderData
            );
        }

        // 解析数据长度（BigEndian格式）
        let data_len = byteorder::ReadBytesExt::read_u32::<BigEndian>(
            &mut &temp[8..model::MIN_DATA_SIZE]
        )?;
        buf.extend_from_slice(&temp[..n]);

        // 循环读取剩余数据
        while buf.len() < model::MIN_DATA_SIZE + data_len as usize {
            let mut tmp_content = utils::get_buffer(data_len as usize);
            let n = conn.read(&mut tmp_content).await?;
            buf.extend_from_slice(&tmp_content[..n]);
            
            if n == 0 { // 处理连接关闭的情况
                return Err(
                    AgentError::PrematureConnectionClose
                );
            }
        }

        Ok(buf)
    }

    pub async fn write(&self) -> Result<(), AgentError> {
        let buf = {
            let guard = self.req.lock().await;
            guard.encode_pack()
        }; // 缩小锁范围
        
        let mut conn_guard = self.conn.lock().await;
        let conn = conn_guard.as_mut().ok_or(AgentError::NotConnected)?; // 明确处理未连接状态
        if let Err(_e) = conn.peer_addr() {
            *conn_guard = None;
            return Err(AgentError::ConnectionLost);
        }

        let mut i = 0;
        while i < buf.len() {
            let n = conn.write(&buf[i..]).await?;
            i += n;
        }
    
        Ok(())
    }

    pub async fn aysnc_worker_do(&self) -> Result<(), Box<dyn std::error::Error+Send+Sync>> {
        let mut cloned_self = self.clone_agent();
        tokio::spawn(async move {
            if let Err(e) = cloned_self.work_loop().await {
                log::error!("worker do error: {}", e);
            }
        });

        Ok(())
    }

    async fn work_loop(&mut self) -> Result<(), Box<dyn std::error::Error+Send+Sync>> {
        loop {
            tokio::select! {
                _ = self.check_cancel() => {
                    break Ok(());
                }
                result = self.read() => {
                    let data = match result {
                        Ok(d) => d,
                        Err(e) => {
                            if self.handle_io_error(&e).await {
                                continue;
                            } else {
                                break Err(Box::new(e));
                            }
                        }
                    };

                    let processed = self.process_data(data).await?;
                    if !processed {
                        break Ok(());
                    }
                }
            }

            tokio::time::sleep(tokio::time::Duration::from_secs(5)).await;
        }
    }

    async fn process_data(&mut self, mut data: Vec<u8>) -> io::Result<bool> {
        // 合并遗留数据
        if !self.left_data.is_empty() {
            data.splice(..0, self.left_data.drain(..));
        }

        // 基础长度校验
        if data.len() < model::MIN_DATA_SIZE {
            self.left_data = data;
            return Ok(true);
        }

        // 解码数据包
        match Response::decode_pack(&data) {
            Ok((resp, processed_len)) => {
                if processed_len != data.len() {
                    self.left_data = data[processed_len..].to_vec();
                } else {
                    self.left_data.clear();
                }

                // 发送到工作队列
                let sender = {
                    let guard = self.worker.lock().await;
                    guard.resps_s.clone()
                };
                sender.send(resp).await.map_err(|_| io::Error::new(ErrorKind::BrokenPipe, "Worker channel closed"))?;
                
                Ok(true)
            }
            Err(e) => match e {
                ResponseError::InsufficientData(_, _) => {
                    self.left_data = data;
                    Ok(true)
                }
                _ => Err(io::Error::new(
                    ErrorKind::InvalidData, 
                    format!("Decode error: {}", e)
                )),
            },
        }
    }

    async fn handle_io_error(&mut self, e: &AgentError) -> bool {
        match e {
            AgentError::Io(io_error) => match io_error.kind() {
                ErrorKind::WouldBlock | ErrorKind::TimedOut => {
                    tokio::time::sleep(model::DIAL_TIME_OUT).await;
                    true
                }
                ErrorKind::ConnectionReset | ErrorKind::BrokenPipe => {
                    if let Ok(_) = self.reconnect().await {
                        true
                    } else {
                        false
                    }
                }
                _ => false
            }
            _ => false
        }
    }

    async fn check_cancel(&self) {
        while !self.cancel_flag.load(Ordering::Relaxed) {
            tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
        }
    }

    pub async fn reconnect(&self) -> Result<(), AgentError> {
        log::info!("Attempting reconnect to {}:{}", self.net, self.addr);
        if let Err(e) = self.connect().await {
            log::error!("Reconnect failed: {}", e);
            Err(AgentError::NotConnected)
        } else {
            Ok(())
        }
    }

    pub async fn heart_beat_ping(&self) -> Result<(), AgentError> {
        {
            let mut guard = self.req.lock().await;
            guard.heart_beat_pack();
        }

        self.write().await
    }

    pub async fn grab(&self) -> Result<(), AgentError> {
        {
            let mut guard = self.req.lock().await;
            guard.grab_data_pack();
        }

        self.write().await
    }

    pub async fn wakeup(&self) -> Result<(), AgentError> {
        {
            let mut guard = self.req.lock().await;
            guard.wakeup_pack();
        }

        self.write().await
    }

    pub async fn del_old_func_msg(&self, func_name: String) -> Result<(), AgentError> {
        {
            let mut guard = self.req.lock().await;
            guard.del_function_pack(func_name);
        }

        if let Err(e) = self.write().await {
            Err(e)
        } else {
            Ok(())
        }
    }

    pub async fn re_add_func_msg(&self, func_name: String) -> Result<(), AgentError> {
        {
            let mut guard = self.req.lock().await;
            guard.del_function_pack(func_name.clone());
        }

        self.write().await
    }

    pub async fn re_set_worker_name(&self, worker_name: String) -> Result<(), AgentError> {
        {
            let mut guard = self.req.lock().await;
            guard.set_worker_name(worker_name);
        }

        self.write().await
    }

    pub fn update_last_time(&self) {
        self.last_time
            .store(chrono::Utc::now().timestamp() as u64, Ordering::Relaxed);
    }

    pub async fn close(&self) -> Result<(), std::io::Error> {
        let mut guard = self.conn.lock().await;
        if let Some(mut stream) = guard.take() {
            stream.shutdown().await
        } else {
            Err(std::io::Error::new(
                std::io::ErrorKind::Other,
                "Connection is not established",
            ))
        }
    }
}