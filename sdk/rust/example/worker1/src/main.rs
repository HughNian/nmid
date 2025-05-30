use std::sync::Arc;
use std::sync::mpsc;
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

fn msgpack_encode_data(data: &models::GetRetStruct) -> Result<Vec<u8>, rmp::encode::ValueWriteError> {
    let mut buf = Vec::new();
    
    // 编码为 map 格式
    rmp::encode::write_map_len(&mut buf, 3)?; // 4 个字段
    
    // code 字段
    rmp::encode::write_str(&mut buf, "Code")?;
    rmp::encode::write_i64(&mut buf, data.code)?; // 强制使用 int64 编码
    
    // msg 字段
    rmp::encode::write_str(&mut buf, "Msg")?;
    rmp::encode::write_str(&mut buf, &data.msg)?;
    
    // data 字段
    rmp::encode::write_str(&mut buf, "Data")?;
    rmp::encode::write_bin(&mut buf, &data.data)?;// 强制使用 int64 编码
    
    Ok(buf)
}

fn to_upper(job: Arc<dyn Job>) -> Result<Vec<u8>, Box<dyn std::error::Error+Send + Sync + 'static>> {
    let resp = job.get_response();
    let value = resp.params_map.get("name").unwrap();
    let upper_value = value.clone().to_uppercase();

    let ret_struct = models::GetRetStruct {
        code: 0,
        msg: "ok".to_string(),
        data: upper_value.into_bytes(),
    };

    let bytes = match msgpack_encode_data(&ret_struct) {
        Ok(data) => data,
        Err(e) => {
            eprintln!("Error encoding data: {}", e);
            return Err(Box::new(e));
        }
    };

    Ok(bytes)
}

#[tokio::main]
async fn main() {
    let worker_name = "Worker1".to_string();
    let server_addr = format!("{}:{}", NMIDSERVERHOST, NMIDSERVERPORT);

    let mut wor = Worker::new();
    wor.set_worker_name(worker_name).add_server(&server_addr).await.unwrap();
    wor.add_function::<fn(Arc<dyn Job>) -> _>("ToUpper", Function::new("ToUpper".to_string(), to_upper)).await.unwrap();

    wor.worker_ready().await.unwrap();

    let wor_clone = wor.worker_clone();

    tokio::spawn(async move {
        wor.worker_do().await;
    });

    let (tx, rx) = mpsc::channel();
    ctrlc::set_handler(move || {
        tx.send(()).unwrap();
    })
    .expect("Error setting Ctrl-C handler");

    rx.recv().unwrap();
    println!("Shutting down...");

    // let (tx, rx) = mpsc::channel();
    // let tx = Arc::new(Mutex::new(tx));
    // ctrlc::set_handler({
    //     let tx = Arc::clone(&tx);
    //     move || {
    //         tx.lock().unwrap().send(()).unwrap();
    //     }
    // })
    // .expect("Error setting Ctrl-C handler");

    // rx.recv().unwrap();

    // println!("Shutting down...");

    wor_clone.worker_close();
}
