pub struct GetRetStruct {
    pub code: u32,
    pub msg: String,
    pub data: Vec<u8>,
}

impl GetRetStruct {
    pub fn new() -> Self {
        GetRetStruct {
            code: 0,
            msg: "".to_string(),
            data: vec![],
        }
    }
}