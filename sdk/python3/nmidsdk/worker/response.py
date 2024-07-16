import struct
import nmidsdk.worker.utils as utils
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
        self.params_map = {}
        self.job_id = ""
        self.job_id_len = 0
        self.ret = bytearray()
        self.ret_len = 0
        self.agent = None

    def decode_pack(self, data: bytearray):
        res_len = len(data)
        if res_len < const.MIN_DATA_SIZE:
            return None, res_len, ConnectionError("Invalid data length")

        cl, = struct.unpack('>I', data[8:const.MIN_DATA_SIZE])
        if res_len < const.MIN_DATA_SIZE + cl:
            return None, res_len, ConnectionError("Invalid data content length")

        content = data[const.MIN_DATA_SIZE:const.MIN_DATA_SIZE+cl]
        if len(content) != cl:
            return None, res_len, ConnectionError("Invalid data content length")

        resp = Response()
        resp.data_type, = struct.unpack('>I', data[4:8])
        resp.data_len = cl
        resp.data = content

        if resp.data_type == const.PDT_S_GET_DATA:
            start = const.MIN_DATA_SIZE
            resp.params_type, = struct.unpack('>I', data[start:start+const.UINT32_SIZE])
            start += const.UINT32_SIZE
            resp.params_handle_type, = struct.unpack('>I', data[start:start+const.UINT32_SIZE])
            start += const.UINT32_SIZE
            resp.handle_len, = struct.unpack('>I', data[start:start+const.UINT32_SIZE])
            start += const.UINT32_SIZE
            resp.params_len, = struct.unpack('>I', data[start:start+const.UINT32_SIZE])
            start += const.UINT32_SIZE
            resp.job_id_len, = struct.unpack('>I', data[start:start+const.UINT32_SIZE])
            
            start += const.UINT32_SIZE
            resp.handle = data[start:start+resp.handle_len].decode('utf-8')
            start += resp.handle_len
            resp.parse_params(data[start:start+resp.params_len])
            start += resp.params_len
            resp.job_id = data[start:start+resp.job_id_len].decode('utf-8')

        return resp, res_len, None
    
    def parse_params(self, params):
        self.params = params
        if self.params_type == const.PARAMS_TYPE_MSGPACK:
            self.params_map = utils.msgpack_params_map(params)
        if self.params_type == const.PARAMS_TYPE_JSON:
            self.params_map = utils.json_params_map(params)

    def get_response(self):
        return self
    
    def should_bind(self, obj: dict):
        obj.update(self.params_map)
        
    def get_params(self):
        if self.params_len == 0:
            return None
        
        return self.params
    
    def get_params_map(self):
        if self.params_len == 0:
            return None
        
        return self.params_map