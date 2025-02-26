use super::{Worker,Request,Response,utils};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use tokio::net::TcpStream;
use tokio::sync::Mutex;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
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
        if let Err(_e) = conn.peer_addr() {
            *conn_guard = None;
            return Err(AgentError::ConnectionLost);
        }

        // 读取初始数据块
        let n = conn.read(&mut temp).await?;
        if n < model::MIN_DATA_SIZE {
            return Err(AgentError::InsufficientHeaderData);
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
                return Err(AgentError::PrematureConnectionClose);
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
        let cloned_self = self.clone_agent();
        tokio::spawn(async move {
            if let Err(e) = cloned_self.work_loop().await {
                log::error!("worker do error: {}", e);
            }
        });

        Ok(())
    }

    async fn work_loop(&self) -> Result<(), Box<dyn std::error::Error+Send+Sync>> {
        loop {
            match self.process_connection().await {
                Ok(_) => self.update_last_time(),
                Err(e) => {
                    log::error!("Connection error: {}", e);
                    self.reconnect().await;
                }
            }

            tokio::time::sleep(tokio::time::Duration::from_secs(5)).await;
        }
    }

    async fn process_connection(&self) -> Result<(), Box<dyn std::error::Error+Send+Sync>> {
        let mut guard = self.conn.lock().await;
        let stream = guard.as_mut().ok_or("Not connected")?;

        // 实现具体的读写逻辑
        // 示例：读取数据到缓冲区
        let mut buf = [0u8; 1024];
        let n = stream.read(&mut buf).await?;
        log::debug!("Received {} bytes", n);

        // 处理协议逻辑
        self.handle_protocol(&buf[..n]).await?;

        Ok(())
    }

    async fn handle_protocol(&self, _data: &[u8]) -> Result<(), Box<dyn std::error::Error+Send+Sync>> {
        // 实现具体的协议解析逻辑
        Ok(())
    }

    pub async fn reconnect(&self) {
        log::info!("Attempting reconnect to {}:{}", self.net, self.addr);
        if let Err(e) = self.connect().await {
            log::error!("Reconnect failed: {}", e);
        }
    }

    pub async fn heart_beat_ping(&self) {
        let mut guard = self.req.lock().await;
        guard.heart_beat_pack();

        let _ = self.write().await;
    }

    pub async fn grab(&self) {
        self.req.lock().await.grab_data_pack();

        let _ = self.write().await;
    }

    pub async fn wakeup(&self) {
        self.req.lock().await.wakeup_pack();

        let _ = self.write().await;
    }

    pub async fn del_old_func_msg(&self, func_name: String) {
        self.req.lock().await.del_function_pack(func_name);

        let _ = self.write().await;
    }

    pub async fn re_add_func_msg(&self, func_name: String) {
        self.req.lock().await.add_function_pack(func_name);

        let _ = self.write().await;
    }

    pub async fn re_set_worker_name(&self, worker_name: String) {
        self.req.lock().await.set_worker_name(worker_name);

        let _ = self.write().await;
    }

    pub fn update_last_time(&self) {
        self.last_time
            .store(chrono::Utc::now().timestamp() as u64, Ordering::Relaxed);
    }

    pub async fn close(&self) {
        let mut guard = self.conn.lock().await;
        if let Some(mut stream) = guard.take() {
            let _ = stream.shutdown().await;
        }
    }
}