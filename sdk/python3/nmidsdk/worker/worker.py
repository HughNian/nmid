import threading
import logging
import time
import asyncio
import nmidsdk.worker.utils as utils
import nmidsdk.model.const as const
from queue import Queue
from nmidsdk.worker.utils import IdGenerator
from nmidsdk.worker.agent import Agent 
from nmidsdk.worker.function import Function, JobFunc
from nmidsdk.worker.response import Response

class Worker:
    def __init__(self):
        self.lock = threading.Lock()
        self.worker_id = ""
        self.worker_name = ""
        self.agents = []
        self.funcs = {}
        self.funcs_num = 0
        self.resps = asyncio.Queue()
        self.ready = False
        self.running = False
        self.use_trace = False
        self.loop = asyncio.get_event_loop()
        self.timer = None

    def set_worker_id(self, wid: str):
        if wid == "":
            self.worker_id = IdGenerator().get_id()
        else:
            self.worker_id = wid
        return self
    
    def set_worker_name(self, wname: str):
        if len(wname) == 0:
            self.worker_name = IdGenerator().get_id() 
        else:
            self.worker_name = wname
        return self

    def get_worker_key(self):
        key = self.worker_name
        if key == "":
            key = self.worker_id
        
        if key == "":
            key = IdGenerator().get_id()

        return key
    
    def add_server(self, net: tuple, addr: str):
        agent = Agent(net, addr, self)
        if agent is None:
            logging.error("agent None")
            return ValueError("agent None")

        self.agents.append(agent)

        return None

    def add_function(self, func_name: str, job_func: JobFunc):
        with self.lock:
            if func_name in self.funcs:
                logging.error("function %s already exist", func_name)
                return ValueError("function %s already exist" % func_name)
            
            self.funcs[func_name] = Function(job_func, func_name)
            self.funcs_num += 1

        if self.running:
            self.msg_broadcast(func_name, const.PDT_W_ADD_FUNC)

    def del_function(self, func_name: str):
        with self.lock:
            if func_name not in self.funcs:
                logging.error("function %s not exist", func_name)
                return ValueError("function %s not exist" % func_name)
            
            self.funcs.pop(func_name)
            self.funcs_num -= 1

        if self.running == True:
            self.msg_broadcast(func_name, const.PDT_W_DEL_FUNC)

    def get_function(self, func_name: str):
        if len(self.funcs) == 0 or self.funcs_num == 0:
            return None, ValueError("worker have no funcs")
        
        self.lock.acquire()
        f = self.funcs.get(func_name)
        self.lock.release()

        if f is None:
            return None, ValueError("not found")
        
        if isinstance(f, Function) and f.func_name != func_name:
            return None, ValueError("not found")
        
        return f, None
    
    def do_function(self, resp: Response):
        if resp.data_type == const.PDT_S_GET_DATA:
            func_name = resp.handle
            function, err = self.get_function(func_name)
            if err is not None:
                return err
            if function is not None:
                if function.func_name != func_name:
                    return ValueError("funcname error")
                elif resp.params_len == 0:
                    return ValueError("params error")
                
                ret, err = function.func(resp)
                if err is None:
                    resp.agent.req.handle_len = resp.handle_len
                    resp.agent.req.handle = resp.handle
                    resp.agent.req.params_len = resp.params_len
                    resp.agent.req.params = resp.params
                    resp.agent.req.job_id_len = resp.job_id_len
                    resp.agent.req.job_id = resp.job_id

                # while resp.agent.lock:
                    resp.agent.req.ret_pack(ret)
                    resp.agent.write()

    def msg_broadcast(self, name: str, flag: int):
        bname = bytearray(name.encode('utf-8'))
        for agent in self.agents:
           if isinstance(agent, Agent):
            if flag == const.PDT_W_SET_NAME:
                agent.req.set_worker_name(bname)
            if flag == const.PDT_W_ADD_FUNC:
                agent.req.add_function_pack(bname)
            if flag == const.PDT_W_DEL_FUNC:
                agent.req.del_function_pack(bname)
            else:
                agent.req.add_function_pack(bname)

            agent.write()

    def worker_ready(self) -> Exception:
        if len(self.agents) == 0:
            return ConnectionError("none active agents")
        
        if self.funcs_num == 0 or len(self.funcs) == 0:
            return ConnectionError("none funcs")
        
        for agent in self.agents:
            if isinstance(agent, Agent):
                e = agent.connect()
                if e is not None:
                    return e
                
        self.msg_broadcast(self.worker_name, const.PDT_W_SET_NAME)

        for func in self.funcs.values():
            if isinstance(func, Function):
                self.msg_broadcast(func.func_name, const.PDT_W_ADD_FUNC)

        self.lock.acquire()
        self.ready = True
        self.lock.release_lock()

    async def process_resp(self):
        while True:
            resp = await self.resps.get()
            if resp is not None and isinstance(resp, Response):
                if resp.data_type == const.PDT_TOSLEEP:
                    time.sleep(2)
                    threading.Thread(target=resp.agent.wakeup).start()
                elif resp.data_type == const.PDT_S_GET_DATA:
                    e = self.do_function(resp)
                    if e is not None:
                        logging.error(e)
                elif resp.data_type == const.PDT_NO_JOB:
                    # threading.Thread(target=resp.agent.grab).start()
                    print("grab here")
                elif resp.data_type == const.PDT_S_HEARTBEAT_PONG:
                    resp.agent.last_time = utils.get_millisecond()
                elif resp.data_type == const.PDT_WAKEUPED:
                    # threading.Thread(target=resp.agent.grab).start()
                    print("grab here")

    def heart_beat(self):
        if self.timer is not None:
            self.timer.cancel()
        
        self.timer = threading.Timer(const.DEFAULTHEARTBEATTIME, self._peform_heart_beat).start()

    def _peform_heart_beat(self):
        for agent in self.agents:
            if isinstance(agent, Agent):
                agent.heart_beat_ping()
        
        if self.running:
            self.heart_beat()
    
    def worker_time_out(self):
        while self.running:
            for agent in self.agents:
                if isinstance(agent, Agent):
                    if utils.get_millisecond() - agent.last_time > const.DEFAULTHEARTBEATTIME * 1000:
                        self.worker_re_connect(agent)

                time.sleep(5)

    def worker_do(self):
        if self.ready == False:
            e = self.worker_ready()
            if e is not None:
                logging.fatal(e)

        self.lock.acquire()
        self.running = True
        self.lock.release()

        threading.Thread(target=self.heart_beat).start()

        threading.Thread(target=self.worker_time_out).start()

        self.loop.run_until_complete(self.process_resp())

    def worker_re_connect(self, agent: Agent):
        for fname, _ in self.funcs.items():
            agent.del_old_func_msg(fname)

        agent.re_connect()
        agent.re_set_worker_name(self.worker_name)
        for fname, _ in self.funcs.items():
            agent.re_add_func_msg(fname)

    def worker_close(self):
        if self.running == True:
            for fn, _ in self.funcs.items():
                self.msg_broadcast(fn, const.PDT_W_DEL_FUNC)

            for agent in self.agents:
                if isinstance(agent, Agent):
                    agent.close()
            
            self.running = False

            if self.timer is not None:
                self.timer.cancel()
                self.timer = None
            
            self.loop.close()