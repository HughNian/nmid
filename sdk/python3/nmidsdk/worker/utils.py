import time
import msgpack
import json
import threading

class IdGenerator:
    def __init__(self):
        self.value_lock = threading.Lock()
        self.next_value = int(time.time() * 10 ** 9) << 32

    def get_id(self):
        with self.value_lock:
            current_time_ns = int(time.time() * 10 ** 9) << 32
            if current_time_ns > self.next_value:
                self.next_value = current_time_ns
            self.next_value += 1
            return str(self.next_value)
        
def msgpack_params_map(params):
    return msgpack.unpackb(params)

def json_params_map(params):
    return json.loads(params)

def get_millisecond():
    return int(time.time() * 1000)

def get_now_second():
    millisecond = threading.atomic.AtomicLong(get_millisecond())
    return millisecond.value // 1000
