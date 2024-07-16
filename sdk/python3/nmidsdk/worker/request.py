import struct
import nmidsdk.model.const as const

class Request:
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
        self.job_id = ""
        self.job_id_len = 0
        self.ret = bytearray()
        self.ret_len = 0

    def heart_beat_pack(self):
        data = "PING"
        self.data_type = const.PDT_W_HEARTBEAT_PING
        self.data_len = len(data)
        self.data = data.encode()
        
        return self.data

    def set_worker_name(self, worker_name: bytearray) -> bytearray:
        self.data_type = const.PDT_W_SET_NAME
        self.data_len = len(worker_name)
        self.data = worker_name

        return self.data
    
    def add_function_pack(self, func_name: bytearray) -> bytearray:
        self.data_type = const.PDT_W_ADD_FUNC
        self.data_len = len(func_name)
        self.data = func_name

        return self.data
    
    def del_function_pack(self, func_name: bytearray) -> bytearray:
        self.data_type = const.PDT_W_DEL_FUNC
        self.data_len = len(func_name)
        self.data = func_name

        return self.data
    
    def grab_data_pack(self) -> bytearray:
        self.data_type = const.PDT_W_GRAB_JOB
        self.data_len = 0
        self.data = bytearray()

        return self.data
    
    def wakeup_pack(self) -> bytearray:
        self.data_type = const.PDT_WAKEUP
        self.data_len = 0
        self.data = bytearray()

        return self.data
    
    def limit_exceed_pack(self) -> bytearray:
        self.data_type = const.PDT_RATELIMIT
        self.data_len = 0
        self.data = bytearray()

        return self.data
    
    def ret_pack(self, ret: bytes) -> bytearray:
        self.ret = ret
        self.ret_len = len(ret)

        self.data_type = const.PDT_W_RETURN_DATA
        self.data_len = const.UINT32_SIZE + self.handle_len + const.UINT32_SIZE + self.params_len + const.UINT32_SIZE + self.ret_len + const.UINT32_SIZE + self.job_id_len

        length = int(self.data_len)
        content = bytearray(length)

        # 序列化数据
        content[:const.UINT32_SIZE] = struct.pack('>I', self.handle_len)
        content[const.UINT32_SIZE:const.UINT32_SIZE + self.handle_len] = self.handle.encode()
        content[const.UINT32_SIZE + self.handle_len:const.UINT32_SIZE + self.handle_len + const.UINT32_SIZE] = struct.pack('>I', self.params_len)
        content[const.UINT32_SIZE + self.handle_len + const.UINT32_SIZE:const.UINT32_SIZE + self.handle_len + const.UINT32_SIZE + self.params_len] = self.params
        content[const.UINT32_SIZE + self.handle_len + const.UINT32_SIZE + self.params_len:const.UINT32_SIZE + self.handle_len + 2*const.UINT32_SIZE + self.params_len] = struct.pack('>I', self.ret_len)
        content[const.UINT32_SIZE + self.handle_len + 2*const.UINT32_SIZE + self.params_len:const.UINT32_SIZE + self.handle_len + 2*const.UINT32_SIZE + self.params_len + self.ret_len] = self.ret
        content[const.UINT32_SIZE + self.handle_len + 2*const.UINT32_SIZE + self.params_len + self.ret_len:const.UINT32_SIZE + self.handle_len + 3*const.UINT32_SIZE + self.params_len + self.ret_len] = struct.pack('>I', self.job_id_len)
        content[const.UINT32_SIZE + self.handle_len + 3*const.UINT32_SIZE + self.params_len + self.ret_len:const.UINT32_SIZE + self.handle_len + 3*const.UINT32_SIZE + self.params_len + self.ret_len + self.job_id_len] = self.job_id.encode()

        self.data = content
        return content

    def encode_pack(self) -> bytearray:
        len = const.MIN_DATA_SIZE + self.data_len  # add 12 bytes head
        data = bytearray(len)

        # 添加头部信息
        data[:4] = struct.pack('>I', const.CONN_TYPE_WORKER)
        data[4:8] = struct.pack('>I', self.data_type)
        data[8:const.MIN_DATA_SIZE] = struct.pack('>I', self.data_len)
        data[const.MIN_DATA_SIZE:] = self.data

        return data