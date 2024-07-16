import socket
import errno
import asyncio
import logging
import threading
import nmidsdk.model.const as const
from nmidsdk.client.request import Request
from nmidsdk.client.response import Response, RespHandlerMap
from typing import Callable, Any

class Client:
    def __init__(self, network, addr):
        self.lock = threading.Lock()
        self.net = network
        self.addr = addr
        self.conn = None
        self.req = None
        self.res = Response()
        self.res_queue = asyncio.Queue()
        self.io_time_out = None
        self.err_handler: Callable[[Any], None]
        self.resp_handlers = RespHandlerMap()
        self.loop = asyncio.get_event_loop()

    def set_io_time_out(self, t):
        self.io_time_out = t
        return self

    def client_conn(self):
        try:
            self.conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self.conn.connect(self.addr)
            self.conn.settimeout(const.DIAL_TIME_OUT)
            self.conn.setblocking(False)
            return None
        except Exception as e:
            return e

    def start(self):
        err = self.client_conn()
        logging.info("Connection error: %s", err)
        if err is not None:
            logging.info("Connection error: %s", err)
            return None

        threading.Thread(target=self.client_read).start()

        return self.conn

    def write(self):
        with self.lock:
            buf = self.req.encode_pack()
            total_sent = 0
            while total_sent < len(buf):
                n = self.conn.send(buf[total_sent:])
                if n == -1:
                    return ValueError("agent None")
                total_sent += n

        return None

    def read(self, length):
        buf = bytearray(length)
        view = memoryview(buf)
        data = bytearray()

        if self.conn == None:
            return None, ConnectionError("connection closed unexpectedly read error")

        while length > 0 or len(data) < const.MIN_DATA_SIZE:
            n = self.conn.recv_into(view, length)
            if n == 0:
                return data, ConnectionError("connection closed unexpectedly read error")
            
            data.extend(view[:n])
            view = view[n:]
            length -= n

            if n < const.MIN_DATA_SIZE:
                break

        return data, None

    def client_read(self):
        data = bytearray()
        left_data = bytearray()
        while self.conn is not None:
            try:
                data, err = self.read(const.MIN_DATA_SIZE)
            except socket.timeout as opErr:
                logging.info("Read operation timed out: %s", opErr)
            except socket.error as opErr:
                if opErr.errno == socket.EAGAIN or opErr.errno == socket.EWOULDBLOCK:
                    continue  # 临时错误，继续读取
                elif opErr.errno == errno.ECONNRESET:  # 可能表示服务器关闭了连接
                    logging.info("Server closed the connection.")
                else:
                    logging.info("Client read error: %s", opErr)
                    self.close()
            except EOFError:
                logging.info("Connection closed by server.")
            except Exception as e:
                logging.error("Unexpected error during read: %s", e)
                break

            if left_data is not None and len(left_data) > 0:
                data = left_data + data #left_data 需在前面
                left_data = None

            while True:
                l = len(data)
                if l < const.MIN_DATA_SIZE:
                    left_data = data
                    break

                if left_data is not None and len(left_data) == 0:
                    conn_type = self.res.get_conn_type(data)
                    if conn_type != const.CONN_TYPE_SERVER:
                        break

                res, res_len, err = self.res.decode_pack(data)
                if err is not None:
                    left_data = data[:res_len]
                    break
                else:
                    self.loop.call_soon_threadsafe(self.loop.create_task, self.res_queue.put(res))

                data = data[l:]
                if len(data) > 0:
                    continue
                else:
                    break

    def handler_resp(self, resp):
        if resp is None:
            return
        
        if len(resp.handle) == 0 or resp.handle_len == 0:
            return

        key = resp.handle
        handler = self.resp_handlers.get_res_handler_map(key)
        if handler is not None:
            handler(resp)
            self.resp_handlers.delete_res_handler_map(key)
            return

    async def process_resp(self):
        res = await self.res_queue.get()
        if res is not None and isinstance(res, Response):
            if res.data_type == const.PDT_ERROR:
                self.err_handler(res.get_res_error())
                return
            elif res.data_type == const.PDT_CANT_DO:
                self.err_handler(res.get_res_error())
                return
            elif res.data_type == const.PDT_RATELIMIT:
                self.err_handler(res.get_res_error())
                return
            elif res.data_type == const.PDT_S_RETURN_DATA:
                self.handler_resp(res)
                return

    def set_params_handle(self, hType):
        if hType != const.PARAMS_HANDLE_TYPE_ENCODE and hType != const.PARAMS_HANDLE_TYPE_ORIGINAL:
            logging.info("set params handle type value error not in encode or original")
            return self
        
        if self.req is None:
            self.req = Request()
        
        self.req.params_handle_type = hType
        return self

    def do(self, func_name: str, params: bytearray, callback: Callable[[Response], None]) -> None:
        if self.conn is None:
            raise ConnectionError("Connection is not established")

        self.resp_handlers.put_res_handler_map(func_name, callback)

        if self.req is None:
            self.req = Request()

        self.req.content_pack(const.PDT_C_DO_JOB, func_name, params)

        err = self.write()
        if err is not None:
            return err
        
        # self.loop.create_task(self.process_resp())
        self.loop.run_until_complete(self.process_resp())
    
    def close(self):
        with self.lock:
            if self.conn is not None:
                self.conn.close()
                self.conn = None
                self.res_queue = None
                self.resp_handlers.holder = {}
                self.resp_handlers = None

            self.loop.stop()
