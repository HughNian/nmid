use super::{Agent,Job};
use serde_json::Value;
use std::collections::HashMap;
use std::sync::Arc;


#[derive(Debug, Clone)]
pub struct Response {
    // 基础数据段
    pub data_type: u32,
    pub data: Vec<u8>,
    pub data_len: u32,

    // 处理标识段
    pub handle: String,
    pub handle_len: u32,
    
    // 参数段
    pub params_type: u32,
    pub params_handle_type: u32,
    pub params_len: u32,
    pub params: Vec<u8>,
    pub params_map: HashMap<String, Value>,  //使用serde_json处理动态类型

    // 任务标识
    pub job_id: String,
    pub job_id_len: u32,

    // 返回结果
    pub ret: Vec<u8>,
    pub ret_len: u32,

    // 关联对象
    pub agent: Option<Arc<Agent>>,  // 使用Arc保证线程安全
}

impl Response {
    pub fn new() -> Self {
        Self {
            data_type: 0,
            data: vec![],
            data_len: 0,
            handle: "".to_string(),
            handle_len: 0,
            params_type: 0,
            params_handle_type: 0,
            params_len: 0,
            params: vec![],
            params_map: HashMap::new(),
            job_id: String::new(),
            job_id_len: 0,
            ret: Vec::new(),
            ret_len: 0,
            agent: None,
        }
    }
}

impl Job for Response {
    fn get_response(&self) -> Self {
        self.clone()
    }
    fn parse_params(&mut self, params: Vec<u8>) {
        
    }
    fn get_params(&self) -> Vec<u8> {
        self.data.clone()
    }
    fn get_params_map(&self) -> HashMap<String, Value> {
        self.params_map.clone()
    }
}