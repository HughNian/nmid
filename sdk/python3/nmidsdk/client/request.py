import struct
import nmidsdk.model.const as const

class Request:
    def __init__(self):
        self.data_type = 0
        self.data = bytearray()
        self.data_len = 0
        self.handle = ""
        self.handle_len = 0
        self.params_type = const.PARAMS_TYPE_MSGPACK
        self.params_handle_type = const.PARAMS_HANDLE_TYPE_ENCODE
        self.params_len = 0
        self.params = bytearray()
        self.ret = bytearray()
        self.ret_len = 0

    def content_pack(self, dataType: int, handle:str, params: bytearray):
        self.data_type = dataType
        self.handle = handle
        self.handle_len = len(handle)
        self.params = params
        self.params_len = len(params)
        self.data_len = const.UINT32_SIZE + const.UINT32_SIZE + const.UINT32_SIZE + self.handle_len + const.UINT32_SIZE + self.params_len
        
        content = bytearray(self.data_len)
        struct.pack_into('>I', content, 0, self.params_type)
        struct.pack_into('>I', content, const.UINT32_SIZE, self.params_handle_type)
        struct.pack_into('>I', content, const.UINT32_SIZE * 2, self.handle_len)
        content[const.UINT32_SIZE * 3:const.UINT32_SIZE * 3 + self.handle_len] = handle.encode()
        struct.pack_into('>I', content, const.UINT32_SIZE * 3 + self.handle_len, self.params_len)
        content[const.UINT32_SIZE * 4 + self.handle_len:const.UINT32_SIZE * 4 + self.handle_len + self.params_len] = params
        
        self.data = content
        return self.data, self.data_len

    def encode_pack(self):
        len = const.MIN_DATA_SIZE + self.data_len
        data = bytearray(len)

        struct.pack_into('>I', data, 0, const.CONN_TYPE_CLIENT)
        struct.pack_into('>I', data, const.UINT32_SIZE, self.data_type)
        struct.pack_into('>I', data, const.UINT32_SIZE * 2, self.data_len)
        data[const.MIN_DATA_SIZE:] = self.data

        return data