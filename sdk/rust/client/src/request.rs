// Client request encoder.
// 对应 node sdk: lib/client/request.js
//
// 包格式 (big-endian):
//   header (12 bytes): ConnType(4) | DataType(4) | DataLen(4)
//   body (DataLen bytes) for PDT_C_DO_JOB:
//     ParamsType(4) | ParamsHandleType(4) | HandleLen(4) | Handle |
//     ParamsLen(4) | Params

use byteorder::{BigEndian, WriteBytesExt};
use std::io::Write;

#[derive(Debug)]
pub struct Request {
    pub data_type: u32,
    pub data: Vec<u8>,
    pub data_len: u32,

    pub handle: String,
    pub handle_len: u32,

    pub params_type: u32,
    pub params_handle_type: u32,
    pub params_len: u32,
    pub params: Vec<u8>,
}

impl Request {
    pub fn new() -> Self {
        Request {
            data_type: 0,
            data: Vec::new(),
            data_len: 0,
            handle: String::new(),
            handle_len: 0,
            params_type: model::PARAMS_TYPE_MSGPACK,
            params_handle_type: model::PARAMS_HANDLE_TYPE_ENCODE,
            params_len: 0,
            params: Vec::new(),
        }
    }

    // 构建 PDT_C_DO_JOB body
    pub fn content_pack(&mut self, data_type: u32, handle: String, params: Vec<u8>) {
        self.data_type = data_type;
        self.handle = handle.clone();
        self.handle_len = handle.len() as u32;
        self.params = params.clone();
        self.params_len = params.len() as u32;

        self.data_len = model::UINT32_SIZE // params_type
            + model::UINT32_SIZE // params_handle_type
            + model::UINT32_SIZE // handle_len
            + self.handle_len
            + model::UINT32_SIZE // params_len
            + self.params_len;

        let mut content = vec![0u8; self.data_len as usize];
        {
            let mut writer = &mut content[..];
            writer.write_u32::<BigEndian>(self.params_type).unwrap();
            writer.write_u32::<BigEndian>(self.params_handle_type).unwrap();
            writer.write_u32::<BigEndian>(self.handle_len).unwrap();
            writer.write_all(self.handle.as_bytes()).unwrap();
            writer.write_u32::<BigEndian>(self.params_len).unwrap();
            writer.write_all(&self.params).unwrap();
        }

        self.data = content;
    }

    // 加 12 字节 header
    pub fn encode_pack(&self) -> Vec<u8> {
        let len = model::MIN_DATA_SIZE + self.data_len as usize;
        let mut data = vec![0u8; len];
        {
            let mut writer = &mut data[..];
            writer.write_u32::<BigEndian>(model::CONN_TYPE_CLIENT).unwrap();
            writer.write_u32::<BigEndian>(self.data_type).unwrap();
            writer.write_u32::<BigEndian>(self.data_len).unwrap();
            writer.write_all(&self.data).unwrap();
        }
        data
    }
}
