use std::time::{SystemTime, UNIX_EPOCH};
pub fn get_buffer(n: usize) -> Vec<u8> {
    vec![0u8; n]
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