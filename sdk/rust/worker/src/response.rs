use super::{Agent,Job,utils};
use std::collections::HashMap;
use std::sync::Arc;
use thiserror::Error;
use byteorder::{BigEndian, ReadBytesExt};
use rmpv::Value as MValue;
use serde_json::Value as JValue;

#[derive(Debug, Error)]
pub enum ResponseError {
    #[error("params pase error")]
    ParamsParseError,
    #[error("unkonw params type")]
    UnknownParamsType,
    #[error("invalid data format: {0}")]
    InvalidData(String),
    #[error("I/O error: {0}")]
    IoError(#[from] std::io::Error),
    #[error("insufficient data: {0},{1}")]
    InsufficientData(String,String),
}

#[derive(Debug, Clone)]
pub enum ParamsValue {
    MsgPackValue(MValue),
    JsonValue(JValue),  
}

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
    pub params_map: HashMap<String, ParamsValue>,

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

    pub fn decode_pack(data: &[u8]) -> Result<(Response, usize), ResponseError> {
        let res_len = data.len();
        if res_len < model::MIN_DATA_SIZE {
            return Err(ResponseError::InvalidData(format!("invalid data: {:?}", data)));
        }

        let cl = ReadBytesExt::read_u32::<BigEndian>(&mut &data[8..model::MIN_DATA_SIZE])? as usize;
        if res_len < model::MIN_DATA_SIZE + cl {
            return Err(ResponseError::InvalidData(format!("invalid data: {:?}", data)));
        }

        let content = &data[model::MIN_DATA_SIZE..model::MIN_DATA_SIZE + cl];
        if content.len() != cl {
            return Err(ResponseError::InvalidData(format!("invalid data: {:?}", data)));
        }

        let mut resp = Response::new();
        resp.data_type = ReadBytesExt::read_u32::<BigEndian>(&mut &data[4..8])?;
        resp.data_len = cl as u32;
        resp.data = content.to_vec();

        if resp.data_type == model::PDT_S_GET_DATA {
            let mut start = model::MIN_DATA_SIZE;
            let mut end = start + model::UINT32_SIZE as usize;
            resp.params_type = ReadBytesExt::read_u32::<BigEndian>(&mut &data[start..end])?;
            start = end;
            end = start + model::UINT32_SIZE as usize;
            resp.params_handle_type = ReadBytesExt::read_u32::<BigEndian>(&mut &data[start..end])?;
            start = end;
            end = start + model::UINT32_SIZE as usize;
            resp.handle_len = ReadBytesExt::read_u32::<BigEndian>(&mut &data[start..end])?;
            start = end;
            end = start + model::UINT32_SIZE as usize;
            resp.params_len = ReadBytesExt::read_u32::<BigEndian>(&mut &data[start..end])?;
            start = end;
            end = start + model::UINT32_SIZE as usize;
            resp.job_id_len = ReadBytesExt::read_u32::<BigEndian>(&mut &data[start..end])?;
            start = end;
            end = start + resp.handle_len as usize;
            resp.handle = String::from_utf8_lossy(&data[start..end]).to_string();
            start = end;
            end = start + resp.params_len as usize;
            resp.parse_params(data[start..end].to_vec());
            start = end;
            end = start + resp.job_id_len as usize;
            resp.job_id = String::from_utf8_lossy(&data[start..end]).to_string();
        }

        Ok((resp, res_len))
    }

    pub fn decode_pack2(data: &[u8]) -> Result<(Response, usize), ResponseError> {
        let res_len = data.len();
        if res_len < model::MIN_DATA_SIZE {
            return Err(ResponseError::InvalidData(format!("invalid data: {:?}", data)));
        }

        let cl = ReadBytesExt::read_u32::<BigEndian>(&mut &data[8..model::MIN_DATA_SIZE])? as usize;
        if res_len < model::MIN_DATA_SIZE + cl {
            return Err(ResponseError::InvalidData(format!("invalid data: {:?}", data)));
        }

        let content = &data[model::MIN_DATA_SIZE..model::MIN_DATA_SIZE + cl];
        if content.len() != cl {
            return Err(ResponseError::InvalidData(format!("invalid data: {:?}", data)));
        }

        let mut resp = Response::new();
        resp.data_type = ReadBytesExt::read_u32::<BigEndian>(&mut &data[4..8])?;
        resp.data_len = cl as u32;
        resp.data = content.to_vec();

        if resp.data_type == model::PDT_S_GET_DATA {
            let mut cursor = model::MIN_DATA_SIZE;  // 使用单个游标替代 start/end
        
            // 辅助宏：读取 u32 并移动游标
            macro_rules! read_u32 {
                () => {{
                    let val = ReadBytesExt::read_u32::<BigEndian>(
                        &mut &data[cursor..cursor + model::UINT32_SIZE as usize]
                    )?;
                    cursor += model::UINT32_SIZE as usize;
                    val
                }};
            }
        
            // 读取所有 u32 字段
            resp.params_type = read_u32!();
            resp.params_handle_type = read_u32!();
            resp.handle_len = read_u32!();
            resp.params_len = read_u32!();
            resp.job_id_len = read_u32!();
        
            // 辅助函数：读取字符串
            let read_string = |cursor: &mut usize, len: u32| -> Result<String, ResponseError> {
                let len = len as usize;
                let s = String::from_utf8_lossy(&data[*cursor..*cursor + len]).into_owned();
                *cursor += len;
                Ok(s)
            };
        
            // 读取字符串和参数
            resp.handle = read_string(&mut cursor, resp.handle_len)?;
            resp.params = data[cursor..cursor + resp.params_len as usize].to_vec();
            resp.parse_params(resp.params.clone());
            cursor += resp.params_len as usize;
            resp.job_id = read_string(&mut cursor, resp.job_id_len)?;
        }

        Ok((resp, res_len))
    }
}

impl Job for Response {
    fn get_response(&self) -> Self {
        self.clone()
    }

    fn parse_params(&mut self, params: Vec<u8>) {
        if self.data_type == model::PARAMS_TYPE_MSGPACK {
            self.params_map = utils::msgpack_params_map(params).unwrap();
        } else if self.data_type == model::PARAMS_TYPE_JSON {
            self.params_map = utils::json_params_map(params).unwrap();
        }
    }

    fn get_params(&self) -> Vec<u8> {
        self.data.clone()
    }
    
    fn get_params_map(&self) -> HashMap<String, ParamsValue> {
        self.params_map.clone()
    }
}