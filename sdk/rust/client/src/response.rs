// Client response decoder.
// 对应 node sdk: lib/client/response.js
//
// PDT_S_RETURN_DATA 的 body (从 packet 的 MIN_DATA_SIZE 处读):
//   HandleLen(4) | ParamsLen(4) | RetLen(4) |
//   Handle | Params | Ret

use byteorder::{BigEndian, ReadBytesExt};
use thiserror::Error;

#[derive(Debug, Error)]
pub enum ClientError {
    #[error("request error")]
    PdtError,
    #[error("have no job do")]
    PdtCantDo,
    #[error("have ratelimit")]
    PdtRatelimit,
    #[error("I/O error: {0}")]
    Io(#[from] std::io::Error),
    #[error("invalid data: {0}")]
    InvalidData(String),
}

impl ClientError {
    // 根据 data_type 返回对应的协议错误；非错误类型返回 None
    pub fn from_data_type(dt: u32) -> Option<ClientError> {
        match dt {
            model::PDT_ERROR => Some(ClientError::PdtError),
            model::PDT_CANT_DO => Some(ClientError::PdtCantDo),
            model::PDT_RATELIMIT => Some(ClientError::PdtRatelimit),
            _ => None,
        }
    }

    pub fn code_str(&self) -> &'static str {
        match self {
            ClientError::PdtError => "PDT_ERROR",
            ClientError::PdtCantDo => "PDT_CANT_DO",
            ClientError::PdtRatelimit => "PDT_RATELIMIT",
            _ => "",
        }
    }
}

#[derive(Debug, Clone)]
pub struct Response {
    pub data_type: u32,
    pub data: Vec<u8>,
    pub data_len: u32,

    pub handle: String,
    pub handle_len: u32,

    pub params_len: u32,
    pub params: Vec<u8>,

    pub ret: Vec<u8>,
    pub ret_len: u32,
}

impl Response {
    pub fn new() -> Self {
        Self {
            data_type: 0,
            data: vec![],
            data_len: 0,
            handle: String::new(),
            handle_len: 0,
            params_len: 0,
            params: vec![],
            ret: Vec::new(),
            ret_len: 0,
        }
    }

    // 解码单个完整 packet (>= MIN_DATA_SIZE + data_len 字节) 为 Response
    pub fn decode_pack(data: &[u8]) -> Result<(Response, usize), ClientError> {
        let res_len = data.len();
        if res_len < model::MIN_DATA_SIZE {
            return Err(ClientError::InvalidData("too short".into()));
        }

        let cl = ReadBytesExt::read_u32::<BigEndian>(&mut &data[8..model::MIN_DATA_SIZE])? as usize;
        if res_len < model::MIN_DATA_SIZE + cl {
            return Err(ClientError::InvalidData("incomplete body".into()));
        }

        let content = &data[model::MIN_DATA_SIZE..model::MIN_DATA_SIZE + cl];
        if content.len() != cl {
            return Err(ClientError::InvalidData("content length mismatch".into()));
        }

        let mut resp = Response::new();
        resp.data_type = ReadBytesExt::read_u32::<BigEndian>(&mut &data[4..8])?;
        resp.data_len = cl as u32;
        resp.data = content.to_vec();

        if resp.data_type == model::PDT_S_RETURN_DATA {
            let mut start = model::MIN_DATA_SIZE;
            let mut end = start + model::UINT32_SIZE as usize;
            resp.handle_len = ReadBytesExt::read_u32::<BigEndian>(&mut &data[start..end])?;
            start = end;
            end = start + model::UINT32_SIZE as usize;
            resp.params_len = ReadBytesExt::read_u32::<BigEndian>(&mut &data[start..end])?;
            start = end;
            end = start + model::UINT32_SIZE as usize;
            resp.ret_len = ReadBytesExt::read_u32::<BigEndian>(&mut &data[start..end])?;
            start = end;
            end = start + resp.handle_len as usize;
            resp.handle = String::from_utf8_lossy(&data[start..end]).to_string();
            start = end;
            end = start + resp.params_len as usize;
            resp.params = data[start..end].to_vec();
            start = end;
            end = start + resp.ret_len as usize;
            resp.ret = data[start..end].to_vec();
        }

        Ok((resp, res_len))
    }

    pub fn get_res_result(&self) -> Option<&[u8]> {
        if self.data_type == model::PDT_S_RETURN_DATA {
            Some(&self.ret)
        } else {
            None
        }
    }
}

// 从 packet 前 4 字节读取 conn type
pub fn get_conn_type(data: &[u8]) -> u32 {
    if data.len() < model::UINT32_SIZE as usize {
        return 0;
    }
    ReadBytesExt::read_u32::<BigEndian>(&mut &data[0..model::UINT32_SIZE as usize]).unwrap_or(0)
}
