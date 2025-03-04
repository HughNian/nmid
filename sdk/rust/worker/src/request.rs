use super::utils;
use std::vec;
use thiserror::Error;
use byteorder::{BigEndian, WriteBytesExt};

#[derive(Debug, Error)]
pub enum RequestError {
    #[error("conversion error")]
    ConversionError,
    #[error("buffer overflow")]
    BufferOverflow,
}

#[derive(Debug)]
pub struct Request {
    data_type: u32,
    data: Vec<u8>,
    data_len: usize,

    pub handle: String,
    pub handle_len: u32,

    pub params_type: u32,
    pub params_len: u32,
    pub params: Vec<u8>,

    pub job_id: String,
    pub job_id_len: u32,
    pub ret: Vec<u8>,
    pub ret_len: u32,
}

impl Request {
    pub fn new() -> Self {
        Request {
            data_type: 0,
            data: Vec::new(),
            data_len: 0,
            handle: String::new(),
            handle_len: 0,
            params_type: 0,
            params_len: 0,
            params: vec![],
            job_id: "".to_string(),
            job_id_len: 0,
            ret: vec![],
            ret_len: 0,
        }
    }

    pub fn heart_beat_pack(&mut self) {
        let data = "PING".to_string();

        self.data_type = model::PDT_W_HEARTBEAT_PING;
        self.data_len = data.len();
        self.data = data.into_bytes().to_vec();
    }

    pub fn grab_data_pack(&mut self) {
        self.data_type = model::PDT_W_GRAB_JOB;
        self.data_len = 0;
        self.data = String::from("").into_bytes().to_vec();
    }

    pub fn wakeup_pack(&mut self) -> Option<Vec<u8>> {
        self.data_type = model::PDT_WAKEUP;
        self.data_len = 0;
        self.data = "".to_string().into_bytes().to_vec();
        let content = self.data.clone();

        Some(content)
    }

    pub fn set_worker_name(&mut self, worker_name: String) -> Option<Vec<u8>> {
        self.data_type = model::PDT_W_SET_NAME;
        self.data_len = worker_name.len();
        self.data = worker_name.into_bytes().to_vec();
        let content = self.data.clone();

        Some(content)
    }

    pub fn add_function_pack(&mut self, func_name: String) -> Option<Vec<u8>> {
        self.data_type = model::PDT_W_ADD_FUNC;
        self.data_len = func_name.len();
        self.data = func_name.into_bytes().to_vec();
        let content = self.data.clone();

        Some(content)
    }

    pub fn del_function_pack(&mut self, func_name: String) -> Option<Vec<u8>> {
        self.data_type = model::PDT_W_DEL_FUNC;
        self.data_len = func_name.len();
        self.data = func_name.into_bytes().to_vec();
        let content = self.data.clone();

        Some(content)
    }

    pub fn ret_pack(&mut self, ret: Vec<u8>) -> Result<Vec<u8>, RequestError> {
        self.ret = ret;
        self.ret_len = self.ret.len().try_into().map_err(|_| RequestError::ConversionError)?;

        self.data_type = model::PDT_W_RETURN_DATA;
        self.data_len = (model::UINT32_SIZE + self.handle_len + model::UINT32_SIZE + self.params_len + model::UINT32_SIZE + self.ret_len + model::UINT32_SIZE + self.job_id_len) as usize;

        let mut content = utils::get_buffer(self.data_len);
        content.write_u32::<BigEndian>(self.handle_len).unwrap();
        content.extend_from_slice(self.handle.as_bytes());
        content.write_u32::<BigEndian>(self.params_len).unwrap();
        content.extend_from_slice(self.params.as_slice());
        content.write_u32::<BigEndian>(self.ret_len).unwrap();
        content.extend_from_slice(self.ret.as_slice());
        content.write_u32::<BigEndian>(self.job_id_len).unwrap();
        content.extend_from_slice(self.job_id.as_bytes());

        self.data = content.clone();
        
        Ok(content)
    }

    pub fn encode_pack(&self) -> Vec<u8> {
        let len = model::MIN_DATA_SIZE + self.data_len;
        let mut data = utils::get_buffer(len);

        data.write_u32::<BigEndian>(model::CONN_TYPE_WORKER).unwrap();
        data.write_u32::<BigEndian>(self.data_type).unwrap();
        data.write_u32::<BigEndian>(self.data_len as u32).unwrap();
        data.extend_from_slice(&self.data.as_slice());

        data
    }
}