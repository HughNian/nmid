use super::{Response,WorkerError};
use std::sync::Arc;
use std::collections::HashMap;
use async_trait::async_trait;
use std::any::Any;

#[async_trait]
pub trait Job: Sync + Send {
    fn get_response(&self) -> Response;
    fn parse_params(&mut self, params: Vec<u8>);
    fn get_params(&self) -> Vec<u8>;
    fn get_params_map(&self) -> HashMap<String, Arc<dyn Any + Send + Sync>>;
}

pub trait JobFunc: Sync + Send {
    fn call(&self, job: Arc<dyn Job>) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>>;
}

impl<F> JobFunc for F
where
    F: Fn(Arc<dyn Job>) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> + 'static + Sync + Send
{
    fn call(&self, job: Arc<dyn Job>) -> Result<Vec<u8>, Box<dyn std::error::Error + Send + Sync>> {
        self(job)
    }
}

// 定义 Function 结构体
pub struct Function<F>
where 
    F: JobFunc + 'static
{
    pub func: Arc<F>,
    pub func_name: String,
}

impl<F> Function<F>
where 
    F: JobFunc + 'static
{
    pub fn new(fname: String, jf: F) -> Self {
        Function {
            func: Arc::new(jf),
            func_name: fname.to_string(),
        }
    }
}

pub trait DynamicJobFunc: Send + Sync {
    fn call(&self, job: Arc<dyn Job>) -> Result<Vec<u8>, WorkerError>;
    fn get_func_name(&self) -> String;
}

impl<F> DynamicJobFunc for Function<F> 
where 
    F: JobFunc + 'static
{
    fn call(&self, job: Arc<dyn Job>) -> Result<Vec<u8>, WorkerError> {
        self.func.call(job).map_err(|e| WorkerError::AgentConnection(e.to_string()))
    }

    fn get_func_name(&self) -> String {
        self.func_name.clone()
    }
}