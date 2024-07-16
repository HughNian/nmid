import logging
import signal
import threading
import msgpack
from nmidsdk.worker.response import Response
from nmidsdk.worker.worker import Worker
from typing import Dict

NMID_SERVER_HOST = "127.0.0.1"
NMID_SERVER_PORT = 6808

def to_upper(job: Response):
    resp = job.get_response()
    if resp is None:
        return None, ValueError("response data error")

    class Params(Dict[str, str]):
        pass

    params = Params()
    job.should_bind(params)

    ret_struct = {}
    ret_struct['Code'] = 0
    ret_struct['Msg'] = "ok"
    ret_struct['Data'] = bytearray(params['name'].upper().encode())
    ret = msgpack.packb(ret_struct)
    resp.ret_len = len(ret)
    resp.ret = ret

    return ret, None

def main():
    wname = "Worker1"

    worker = Worker()
    worker.set_worker_name(wname)
    server_addr = (NMID_SERVER_HOST,NMID_SERVER_PORT)
    err = worker.add_server("tcp", server_addr)
    if err:
        logging.fatal(err)
        worker.worker_close()
        return

    worker.add_function("ToUpper", to_upper)

    err = worker.worker_ready()
    if err is not None:
        logging.fatal(err)
        worker.worker_close()
        return

    worker.worker_do()

    quits = threading.Event()
    signal.signal(signal.SIGQUIT, lambda sig, frame: quits.set())
    signal.signal(signal.SIGTERM, lambda sig, frame: quits.set())
    signal.signal(signal.SIGINT, lambda sig, frame: quits.set())

    quits.wait()
    worker.worker_close()

if __name__ == "__main__":
    main()