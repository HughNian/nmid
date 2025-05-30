#[derive(serde::Serialize)]
pub struct GetRetStruct {
    #[serde(rename = "Code")]
    pub code: i64,
    #[serde(rename = "Msg")]
    pub msg: String,
    #[serde(rename = "Data")]
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