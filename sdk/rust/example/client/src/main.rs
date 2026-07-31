// nmid client example.
// 对应 node sdk: example/client/testclient.js
//
// 先启动 nmid server (127.0.0.1:6808) 和暴露 "ToUpper" 的 worker，然后：
//   cargo run -p client1

use client::{Client, ClientError, Response};
use rmpv::Value;

const SERVER_HOST: &str = "127.0.0.1";
const SERVER_PORT: u16 = 6808;

// msgpack 编码 {name: "nihaonihao"}
fn encode_params(name: &str) -> Vec<u8> {
    let v = Value::Map(vec![(Value::from("name"), Value::from(name))]);
    let mut buf = Vec::new();
    rmpv::encode::write_value(&mut buf, &v).expect("msgpack encode failed");
    buf
}

// msgpack 解码 ret 为 {Code, Msg, Data}
fn decode_ret(ret: &[u8]) -> Option<(i64, String, Vec<u8>)> {
    let mut cursor = &ret[..];
    let v = rmpv::decode::read_value(&mut cursor).ok()?;
    let pairs = match v {
        Value::Map(pairs) => pairs,
        _ => return None,
    };

    let mut code = 0i64;
    let mut msg = String::new();
    let mut data = Vec::new();
    for (k, val) in pairs {
        let key = match k {
            Value::String(ref s) => s.as_str()?.to_string(),
            _ => continue,
        };
        match key.as_str() {
            "Code" => code = val.as_i64().unwrap_or(0),
            "Msg" => {
                if let Value::String(ref s) = val {
                    msg = s.as_str().unwrap_or("").to_string();
                }
            }
            "Data" => match val {
                // bin 类型优先
                Value::Binary(b) => data = b,
                // 兼容部分实现把 bin 编码成 str
                Value::String(ref s) => data = s.as_str().unwrap_or("").as_bytes().to_vec(),
                _ => {}
            },
            _ => {}
        }
    }
    Some((code, msg, data))
}

// PDT_CANT_DO 是瞬时错误（worker 可能还在注册或重连），重试几次
async fn call_with_retry(
    client: &Client,
    func_name: &str,
    params: Vec<u8>,
    retries: usize,
    delay_ms: u64,
) -> Result<Response, ClientError> {
    let mut last_err: Option<ClientError> = None;
    for i in 0..retries {
        match client.do_async(func_name, params.clone()).await {
            Ok(resp) => return Ok(resp),
            Err(e) => {
                if matches!(e, ClientError::PdtCantDo) && i < retries - 1 {
                    tokio::time::sleep(tokio::time::Duration::from_millis(delay_ms)).await;
                    continue;
                }
                last_err = Some(e);
            }
        }
    }
    Err(last_err.unwrap_or_else(|| ClientError::InvalidData("call failed".into())))
}

fn handle_response(resp: &Response) {
    let ret = match resp.get_res_result() {
        Some(r) if !r.is_empty() => r,
        _ => {
            return;
        }
    };

    match decode_ret(ret) {
        Some((code, msg, data)) => {
            if code != 0 {
                println!("{}", msg);
            } else {
                println!("{}", String::from_utf8_lossy(&data));
            }
        }
        None => {
            eprintln!("failed to decode ret");
        }
    }
}

#[tokio::main]
async fn main() {
    let addr = format!("{}:{}", SERVER_HOST, SERVER_PORT);
    let client = Client::new(&addr);

    if let Err(e) = client.start().await {
        eprintln!("Error starting client: {}", e);
        return;
    }

    let params = encode_params("nihaonihao");

    match call_with_retry(&client, "ToUpper", params, 5, 300).await {
        Ok(resp) => handle_response(&resp),
        Err(e) => {
            if matches!(e, ClientError::PdtCantDo) {
                eprintln!("have no job do (transient, will retry)");
            } else {
                eprintln!("Client error: {}", e);
            }
        }
    }

    client.close().await;
}
