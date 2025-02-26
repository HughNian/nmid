use std::{fmt::Debug, sync::{atomic::{AtomicBool, Ordering}, Arc, RwLock}};
use std::collections::HashMap;
use std::any::{TypeId, Any};
use tokio::sync::{Mutex,mpsc};
use thiserror::Error;

pub mod agent;
pub mod request;
pub mod response;
pub mod function;
pub mod utils;

pub use request::*;
pub use response::*;
pub use agent::*;
pub use function::*;

#[derive(Debug, Error)]
pub enum WorkerError {
    #[error("no active agents")]
    NoAgents,
    #[error("no registered functions")]
    NoFunctions,
    #[error("agent connection failed: {0}")]
    AgentConnection(String),
}

#[derive(Debug)]
struct WorkerInner {
    agents: Arc<Vec<Arc<Agent>>>,
    funcs: HashMap<String, (TypeId, Arc<dyn Any + Send + Sync>)>,
    funcs_num: usize,
}

#[derive(Debug)]
pub struct Worker {
    worker_id: String,
    worker_name: String,

    inner: Arc<Mutex<WorkerInner>>,
    
    // 原子操作字段
    ready: Arc<RwLock<bool>>,
    running: Arc<AtomicBool>,
    use_trace: Arc<AtomicBool>,
    
    resps: Arc<Mutex<mpsc::Receiver<Response>>>,
}

impl Worker {
    pub fn new() -> Self {
        Worker {
            worker_id: String::new(),
            worker_name: String::new(),
            inner: Arc::new(Mutex::new(WorkerInner {
                agents: Arc::new(Vec::new()),
                funcs: HashMap::new(),
                funcs_num: 0
            })),
            ready: Arc::new(RwLock::new(false)),
            running: Arc::new(AtomicBool::new(false)),
            use_trace: Arc::new(AtomicBool::new(false)),
            resps:  Arc::new(Mutex::new(mpsc::channel(model::QUEUE_SIZE).1))
        }
    }

    // 创建线程安全的克隆
    fn clone_worker(&self) -> Self {
        Self {
            worker_id: self.worker_id.clone(),
            worker_name: self.worker_name.clone(),
            inner: Arc::clone(&self.inner),
            ready: self.ready.clone(),
            running: self.running.clone(),
            use_trace: self.use_trace.clone(),
            resps: Arc::clone(&self.resps),
        }
    }

    pub async fn msg_broadcast(&self, name: String, flag: u32) {
        let guard = self.inner.lock().await;

        for agent in guard.agents.iter() {
            match flag {
                model::PDT_W_SET_NAME => {
                    agent.req.lock().await.set_worker_name(name.clone());
                }
                model::PDT_W_ADD_FUNC => {
                    agent.req.lock().await.add_function_pack(name.clone());
                }
                model::PDT_W_DEL_FUNC => {
                    agent.req.lock().await.del_function_pack(name.clone());
                }
                _=>{
                    agent.req.lock().await.add_function_pack(name.clone());
                }
            }
        }
    }

    pub async fn worker_ready(&self) -> Result<(), WorkerError> {
        let guard = self.inner.lock().await;
        
        if guard.agents.is_empty() {
            return Err(WorkerError::NoAgents);
        }

        if guard.funcs.is_empty() || guard.funcs_num == 0 {
            return Err(WorkerError::NoFunctions);
        }

        // 异步连接所有Agent
        let connection_results = futures::future::join_all(
            guard.agents.iter().map(|agent| async {
                agent.connect()
                    .await
                    .map_err(|e| WorkerError::AgentConnection(e.to_string()))
            })
        ).await;

        // 检查连接结果
        for result in connection_results {
            result?; // 遇到第一个错误立即返回
        }

        // 广播Worker名称
        self.msg_broadcast(
            self.worker_name.clone(), 
            model::PDT_W_SET_NAME
        ).await;

        // 广播所有函数
        for func_name in guard.funcs.keys() {
            self.msg_broadcast(
                func_name.to_string(), 
                model::PDT_W_ADD_FUNC
            ).await;
        }

        // 更新就绪状态
        let mut ready = self.ready.write().unwrap();
        *ready = true;

        Ok(())
    }

    pub async fn worker_do(&mut self) {
        // 确保worker就绪
        if !*self.ready.read().unwrap() {
            if let Err(_) = self.worker_ready().await {
                return;
            }
        }

        // 设置运行状态
        self.running.store(true, Ordering::SeqCst);

        // 启动超时检测任务
        let mut timeout_worker = self.clone_worker();
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(tokio::time::Duration::from_secs(5));
            loop {
                interval.tick().await;
                
                if timeout_worker.running.load(Ordering::SeqCst) == false {
                    break;
                }
                
                let agents = {
                    let guard = timeout_worker.inner.lock().await;
                    guard.agents.clone()
                };

                for agent in agents.iter() {
                    let last_time = agent.last_time.load(Ordering::Relaxed);
                    let now = utils::get_now_second();
                    
                    if now - last_time > model::NMID_SERVER_TIMEOUT {
                        log::error!(
                            "nmid server timeout: server@{} worker@{}",
                            agent.addr,
                            timeout_worker.worker_name
                        );
                        
                        timeout_worker.worker_re_connect(agent.clone()).await;
                    }
                }
            }
        });

        // 启动心跳和任务抓取
        let heartbeat_worker = self.clone_worker();
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(tokio::time::Duration::from_secs(model::DEFAULTHEARTBEATTIME));
            loop {
                interval.tick().await;
                
                if heartbeat_worker.running.load(Ordering::SeqCst) == false {
                    break;
                }
                
                let guard = heartbeat_worker.inner.lock().await;
                for agent in guard.agents.iter() {
                    agent.heart_beat_ping().await;
                    agent.grab().await;
                }
            }
        });

        // 处理响应队列
        while let Some(resp) = {
            let mut guard = self.resps.lock().await;
            guard.recv().await
        } {
            match resp.data_type {
                model::PDT_TOSLEEP => {
                    tokio::time::sleep(tokio::time::Duration::from_secs(2)).await;
                    if let Some(agent) = &resp.agent {
                        agent.wakeup().await;
                    }
                }
                model::PDT_S_GET_DATA => {
                    if let Err(e) = self.do_function(resp).await {
                        log::error!("do function error: {}", e);
                    }
                }
                model::PDT_NO_JOB | model::PDT_WAKEUPED => {
                    if let Some(agent) = &resp.agent {
                        agent.grab().await;
                    }
                }
                model::PDT_S_HEARTBEAT_PONG => {
                    if let Some(agent) = &resp.agent {
                        agent.update_last_time();
                    }
                }
                _ => {
                    if let Some(agent) = &resp.agent {
                        agent.grab().await;
                    }
                }
            }
        }
    }

    pub async fn add_function<F>(&mut self, func_name: &str, func: Function<F>) -> Result<(), WorkerError> 
    where 
        F: JobFunc + 'static
    {
        let mut guard = self.inner.lock().await;
        let type_id = TypeId::of::<F>();
        let dyn_func = Arc::new(func) as Arc<dyn DynamicJobFunc>;
        let vfunc = Arc::new(dyn_func) as Arc<dyn Any + Send + Sync>;

        guard.funcs.insert(func_name.to_string(), (type_id, vfunc));
        guard.funcs_num += 1;

        Ok(())
    }

    pub async fn get_function(&mut self, func_name: &str) -> Result<Arc<dyn DynamicJobFunc>, WorkerError> 
    {
        let guard = self.inner.lock().await;
        if guard.funcs.is_empty() || guard.funcs_num == 0 {
            return Err(WorkerError::NoFunctions);
        }

        guard.funcs.get(func_name)
        .and_then(|(_, any_func)| {
            any_func.downcast_ref::<Arc<dyn DynamicJobFunc>>()
            .map(|f| f.clone())
        })
        .ok_or(WorkerError::NoFunctions)
    }

    pub async fn del_function(&mut self, func_name: &str) -> Result<(), WorkerError> {
        let mut guard = self.inner.lock().await;

        if guard.funcs.is_empty() || guard.funcs_num == 0 {
            return Err(WorkerError::NoFunctions);
        }

        if !guard.funcs.contains_key(func_name) {
            return Err(WorkerError::NoFunctions);
        }

        match guard.funcs.remove(func_name) {
            Some(_) => {
                guard.funcs_num -= 1;

                let agents = guard.agents.clone();
                let func_name = func_name.to_string();
                tokio::spawn(async move {
                    for agent in agents.iter() {
                        agent.req.lock().await.del_function_pack(func_name.clone());
                    }
                });

                Ok(())
            }

            None => {
                return Err(WorkerError::NoFunctions);
            }
        }
    }

    pub async fn do_function(&mut self, resp: Response) -> Result<(), WorkerError>
    {
        if resp.data_type == model::PDT_S_GET_DATA {
            let afunc = self.get_function(&resp.handle).await?;

            let job_instance = Arc::new(resp.clone()) as Arc<dyn Job>;
            let result = afunc.call(job_instance).unwrap();

            if let Some(agent) = &resp.agent {
                {
                    let mut req_guard = agent.req.lock().await;
                    let _ = req_guard.ret_pack(result);
                }

                let _ = agent.write().await;
            }
        }

        Ok(())
    }

    pub async fn worker_re_connect(&mut self, agent: Arc<Agent>) {
        let guard = self.inner.lock().await;

        for func_name in guard.funcs.keys() {
            agent.del_old_func_msg(func_name.clone()).await;
        }

        agent.close().await;

        agent.reconnect().await;

        agent.re_set_worker_name(self.worker_name.clone()).await;

        for func_name in guard.funcs.keys() {
            agent.re_add_func_msg(func_name.clone()).await;
        }
    }
}