import struct
import threading
import nmidsdk.model.const as const

class Response:
    def __init__(self):
        self.data_type = 0
        self.data = bytearray()
        self.data_len = 0
        self.handle = ""
        self.handle_len = 0
        self.params_type = 0
        self.params_handle_type = 0
        self.params_len = 0
        self.params = bytearray()
        self.ret = bytearray()
        self.ret_len = 0

    def get_res_error(self):
        if self.data_type == const.PDT_ERROR:
            return "request error"
        elif self.data_type == const.PDT_CANT_DO:
            return "have no job do"
        elif self.data_type == const.PDT_RATELIMIT:
            return "have ratelimit"
        else:
            return None

    def get_res_result(self):
        if self.data_type == 4:
            return self.ret, None
        else:
            return None, ValueError("data nil")

    def get_conn_type(self, data: bytearray) -> int:
        if len(data) == 0:
            return 0
    
        if len(data) < 4:
            return 0
    
        conn_type, = struct.unpack('>I', data[:const.UINT32_SIZE])
        
        return conn_type

    def decode_pack(self, data: bytearray):
        res_len = len(data)
        if res_len < const.MIN_DATA_SIZE:
            return None, res_len, "Invalid data1"

        cl, = struct.unpack('>I', data[8:const.MIN_DATA_SIZE])
        if res_len < const.MIN_DATA_SIZE + cl:
            return None, res_len, "Invalid data2"

        content = data[const.MIN_DATA_SIZE:const.MIN_DATA_SIZE + cl]
        if len(content) != cl:
            return None, res_len, "Invalid data3"

        resp = Response()
        resp.data_type, = struct.unpack('>I', data[4:8])
        resp.data_len = cl
        resp.data = content

        if resp.data_type == const.PDT_S_RETURN_DATA:
            start = const.MIN_DATA_SIZE
            end = start + const.UINT32_SIZE
            resp.handle_len = struct.unpack('>I', data[start:end])[0]
            start = end
            end = start + const.UINT32_SIZE
            resp.params_len = struct.unpack('>I', data[start:end])[0]
            start = end
            end = start + const.UINT32_SIZE
            resp.ret_len = struct.unpack('>I', data[start:end])[0]
            start = end
            end = start + resp.handle_len
            resp.handle = data[start:end].decode('utf-8')
            start = end
            end = start + resp.params_len
            resp.params = data[start:end]
            start = end
            end = start + resp.ret_len
            resp.ret = data[start:end]

        return resp, res_len, None

class RespHandlerMap:
    def __init__(self):
        self.lock = threading.Lock()
        self.halder = {}

    def put_res_handler_map(self, key, handler):
        with self.lock:
            self.halder[key] = handler

    def get_res_handler_map(self, key):
        with self.lock:
            return self.halder.get(key, None)

    def delete_res_handler_map(self, key):
        with self.lock:
            if key in self.halder:
                del self.halder[key]
