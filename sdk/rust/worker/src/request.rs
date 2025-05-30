use super::utils;
use std::{io::Write, vec};
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
    data_len: u32,

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
        self.data_len = data.len() as u32;
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
        self.data_len = worker_name.len() as u32;
        self.data = worker_name.into_bytes().to_vec();
        let content = self.data.clone();

        Some(content)
    }

    pub fn add_function_pack(&mut self, func_name: String) -> Option<Vec<u8>> {
        self.data_type = model::PDT_W_ADD_FUNC;
        self.data_len = func_name.len() as u32;
        self.data = func_name.into_bytes().to_vec();
        let content = self.data.clone();

        Some(content)
    }

    pub fn del_function_pack(&mut self, func_name: String) -> Option<Vec<u8>> {
        self.data_type = model::PDT_W_DEL_FUNC;
        self.data_len = func_name.len() as u32;
        self.data = func_name.into_bytes().to_vec();
        let content = self.data.clone();

        Some(content)
    }

    pub fn ret_pack(&mut self, ret: Vec<u8>) -> Result<Vec<u8>, RequestError> {
        self.ret = ret;
        self.ret_len = self.ret.len().try_into().map_err(|_| RequestError::ConversionError)?;

        self.data_type = model::PDT_W_RETURN_DATA;
        self.data_len = model::UINT32_SIZE + self.handle_len + model::UINT32_SIZE + self.params_len + model::UINT32_SIZE + self.ret_len + model::UINT32_SIZE + self.job_id_len;

        let mut content = utils::get_buffer(self.data_len as usize);
        let mut writer = &mut content[..]; // 使用切片避免扩容
        writer.write_u32::<BigEndian>(self.handle_len).unwrap();
        writer.write_all(self.handle.as_bytes()).unwrap();
        writer.write_u32::<BigEndian>(self.params_len).unwrap();
        writer.write_all(&self.params).unwrap();
        writer.write_u32::<BigEndian>(self.ret_len).unwrap();
        writer.write_all(&self.ret).unwrap();
        writer.write_u32::<BigEndian>(self.job_id_len).unwrap();
        writer.write_all(self.job_id.as_bytes()).unwrap();

        self.data = content.clone();
        
        Ok(content)
    }

    pub fn ret_pack_manual(&mut self, ret: Vec<u8>) -> Result<Vec<u8>, RequestError> {
        self.ret = ret;
        self.ret_len = self.ret.len() as u32;

        self.data_type = model::PDT_W_RETURN_DATA;
        self.data_len = model::UINT32_SIZE + self.handle_len + 
                        model::UINT32_SIZE + self.params_len + 
                        model::UINT32_SIZE + self.ret_len + 
                        model::UINT32_SIZE + self.job_id_len;

        let mut content = utils::get_buffer(self.data_len as usize);
        let mut start = 0;
        let mut end;

        // Handle length
        let handle_len_bytes = self.handle_len.to_be_bytes();
        end = start + model::UINT32_SIZE as usize;
        content[start..end].copy_from_slice(&handle_len_bytes);
        
        // Handle data
        start = end;
        end = start + self.handle_len as usize;
        content[start..end].copy_from_slice(self.handle.as_bytes());
        
        // Params length
        start = end;
        end = start + model::UINT32_SIZE as usize;
        let params_len_bytes = self.params_len.to_be_bytes();
        content[start..end].copy_from_slice(&params_len_bytes);
        
        // Params data
        start = end;
        end = start + self.params_len as usize;
        content[start..end].copy_from_slice(&self.params);
        
        // Ret length
        start = end;
        end = start + model::UINT32_SIZE as usize;
        let ret_len_bytes = self.ret_len.to_be_bytes();
        content[start..end].copy_from_slice(&ret_len_bytes);
        
        // Ret data
        start = end;
        end = start + self.ret_len as usize;
        content[start..end].copy_from_slice(&self.ret);
        
        // JobId length
        start = end;
        end = start + model::UINT32_SIZE as usize;
        let job_id_len_bytes = self.job_id_len.to_be_bytes();
        content[start..end].copy_from_slice(&job_id_len_bytes);
        
        // JobId data
        start = end;
        end = start + self.job_id_len as usize;
        content[start..end].copy_from_slice(self.job_id.as_bytes());

        self.data = content.clone();
        
        Ok(content)
    }

    pub fn encode_pack(&self) -> Vec<u8> {
        let len = model::MIN_DATA_SIZE + self.data_len as usize;
        let mut data = utils::get_buffer(len);

        let mut writer = &mut data[..]; // 使用切片避免扩容
        writer.write_u32::<BigEndian>(model::CONN_TYPE_WORKER).unwrap();
        writer.write_u32::<BigEndian>(self.data_type).unwrap();
        writer.write_u32::<BigEndian>(self.data_len as u32).unwrap();
        writer.write_all(&self.data).unwrap();

        data
    }
}