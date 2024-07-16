import threading
import socket
import struct
import logging
import time
import nmidsdk.worker.utils as utils
import nmidsdk.model.const as const
from typing import Tuple
from io import BytesIO
from nmidsdk.worker.request import Request
from nmidsdk.worker.response import Response

class Agent:
    def __init__(self, network: str, addr: tuple, w):
        self.lock = threading.Lock()
        self.net = network
        self.addr = addr
        self.conn = None
        self.worker = w
        self.req = Request()
        self.res = Response()
        self.last_time = utils.get_millisecond()

    def connect(self):
        try:
            self.conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self.conn.connect(self.addr)
            self.conn.settimeout(const.DIAL_TIME_OUT)
            self.conn.setblocking(False)
        except Exception as e:
            return e
        
        threading.Thread(target=self.work).start()

    def re_connect(self):
        with self.lock:
            if self.conn is not None:
                self.conn.close()
            
            self.conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            result = self.conn.connect_ex(self.addr)
            if result != 0:
                self.conn = None
                return ConnectionError("connection closed unexpectedly re_connect error")
            
            self.conn.settimeout(const.DIAL_TIME_OUT)
            self.conn.setblocking(False)
            self.last_time = utils.get_millisecond()

    def del_old_func_msg(self, func_name: str):
        self.req.del_function_pack(func_name.encode())
        self.write()

    def re_add_func_msg(self, func_name: str):
        self.req.add_function_pack(func_name.encode())
        self.write()

    def re_set_worker_name(self, worker_name: str):
        self.req.set_worker_name(worker_name.encode())
        self.write()

    def read(self):
        with self.lock:
            if self.conn is None:
                return None, ConnectionError("connection none read error")

            temp = bytearray(const.MIN_DATA_SIZE)
            view = memoryview(temp)
            buf = BytesIO()

            if self.conn == None:
                return None, ConnectionError("connection closed unexpectedly read error")

            n = self.conn.recv_into(view, const.MIN_DATA_SIZE)
            if n == 0:
                return bytearray(), ConnectionError("connection closed read error")
            
            data_len, = struct.unpack('>I', temp[8:const.MIN_DATA_SIZE])
            buf.write(view[:n])
            view = view[n:]

            while buf.tell() < const.MIN_DATA_SIZE + data_len:
                tempcontent = bytearray(data_len)
                viewcontent = memoryview(tempcontent)
                n = self.conn.recv_into(viewcontent, data_len)
                if n == 0:
                    return bytearray(), ConnectionError("connection closed read error")
                
                buf.write(viewcontent[:n])
                viewcontent = viewcontent[n:]

            return bytearray(buf.getvalue()), None

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

    def work(self):
        left_data = bytearray()
        while self.conn is not None:
            try:
                data, err = self.read()
                if err is not None:
                    logging.error("Error reading data: %s", err)
                    continue
            except socket.timeout as opErr:
                logging.info("Read operation timed out: %s", opErr)
            except socket.error as opErr:
                if opErr.errno == socket.EAGAIN or opErr.errno == socket.EWOULDBLOCK:
                    continue  # 临时错误，继续读取
                elif opErr.errno == socket.ECONNRESET:  # 可能表示服务器关闭了连接
                    logging.info("Server closed the connection.")
                    break
                else:
                    logging.info("Worker read error: %s", opErr)
                    continue
            except EOFError:
                logging.info("Connection closed by server.")
                break
            except Exception as e:
                logging.error("Unexpected error during read: %s", e)
                break

            if len(left_data) > 0:
                data = left_data + data

            if len(data) < const.MIN_DATA_SIZE:
                left_data = data
                continue
            
            resp, l, err = self.res.decode_pack(data)
            if err is not None:
                left_data = data
                continue
            elif l != len(data):
                left_data = data
                continue
            else:
                left_data = bytearray()

                if isinstance(resp, Response):
                    resp.agent = self
                
                self.worker.loop.call_soon_threadsafe(self.worker.loop.create_task, self.worker.resps.put(resp))

    def heart_beat_ping(self):
        self.req.heart_beat_pack()
        self.write()
    
    def grab(self):
        self.req.grab_data_pack()
        self.write()

    def wakeup(self):
        self.req.wakeup_pack()
        self.write()

    def limit_exceed(self):
        self.req.limit_exceed_pack()
        self.write()
    def close(self):
        if self.conn is not None:
            self.conn.close()