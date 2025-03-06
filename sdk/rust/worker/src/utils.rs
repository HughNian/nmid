use super::ParamsValue;
use std::time::{SystemTime, UNIX_EPOCH};
use std::collections::HashMap;
use rmpv::{Value, encode};
use serde_json::{from_slice, Value as JValue};
pub fn get_buffer(n: usize) -> Vec<u8> {
    vec![0u8; n]
}

pub fn current_timestamp() -> u64 {
    chrono::Utc::now().timestamp() as u64
}

pub fn get_now_second() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("Time went backwards")
        .as_secs()
}

pub fn get_now_millis() -> u128 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_millis() // 返回 u128
}

pub fn get_now_micros() -> u128 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_micros()
}

pub fn msg_pack_encode(v: Value) -> Vec<u8> {
    let mut buf = Vec::new();
    encode::write_value(&mut buf, &v).unwrap();

    buf
}

pub fn msg_pack_decode(buf: Vec<u8>) -> Value {
    rmpv::decode::read_value(&mut &buf[..]).unwrap()
}

pub fn msgpack_params_map(buf: Vec<u8>) -> Option<HashMap<String, ParamsValue>> {
    let mut params_map:HashMap<String, ParamsValue> = HashMap::new();
    
    match msg_pack_decode(buf) {
        Value::Map(mapkv) => {
            for (key, value) in mapkv {
                match key {
                    Value::String(key_str) => {
                        let map_key = key_str.as_str().unwrap().to_string();
                        params_map.insert(map_key, ParamsValue::MsgPackValue(value));
                    }

                    _ => {
                        log::warn!("unsupported key type: {:?}", key);
                        continue;
                    }
                }
            }
        }
        
        _ => {
            return None;
        }
    }

    Some(params_map)
}

pub fn json_params_map(buf: Vec<u8>) -> Option<HashMap<String, ParamsValue>> {
    let mut params_map: HashMap<String, ParamsValue>   = HashMap::new();

    match from_slice::<JValue>(&buf) {
        Ok(JValue::Object(map)) => {
            for (k, v) in map {
                params_map.insert(k, ParamsValue::JsonValue(v));
            }

            Some(params_map)
        }

        Ok(_) => None, // 非对象类型返回None

        Err(e) => {
            log::warn!("JSON parse error: {}", e);
            None
        }
    }
}