use std::sync::Arc;
use worker::*;
use model::*;
use thiserror::Error;

const NMIDSERVERHOST: &str = "127.0.0.1";
const NMIDSERVERPORT: &str = "6808";

#[derive(Debug, Error)]
pub enum ToUpperError {
    #[error("response data error")]
    ResponseDataError,
    #[error("invalid params: {0}")]
    InvalidParams(String),
}

fn to_upper(job: Arc<dyn Job>) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
    let resp = job.get_response();

    let ret_struct = models::GetRetStruct {
        code: 0,
        msg: "ok".to_string(),
        data: resp.data,
    };

}

#[tokio::main]
async fn main() {
    let worker_name = "Worker1".to_string();
    let server_addr = format!("{}:{}", NMIDSERVERHOST, NMIDSERVERPORT);

    let mut wor = Worker::new();
    wor.set_worker_name(worker_name).add_server(&server_addr);
    wor.add_function("ToUpper", to_upper).await;

    wor.worker_ready();

    wor.worker_do();
}
