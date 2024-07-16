import logging
import msgpack
import nmidsdk.model.const as const
from nmidsdk.client.client import *

SERVER_HOST = "127.0.0.1"
SERVER_PORT = 6808

def main():
    client = None
    try:
        server_addr = (SERVER_HOST,SERVER_PORT)
        client = Client("tcp", server_addr)
        client_conn = client.start()
        if client_conn is None:
            logging.error("Failed to create client")
            return
    except Exception as e:
        logging.error(f"Error starting client: {e}")
        return

    client.err_handler = err_handler

    params_name1 = {"name": "nihaonihao"}
    params1 = msgpack.packb(params_name1)

    # resp_handler = lambda resp: handle_response(resp)
    client.do("ToUpper", bytearray(params1), handle_response)
    client.close()

def err_handler(e):
    if e == const.RESTIMEOUT:
        logging.info("Time out here")
    else:
        logging.error(e)
    
    print("Client error here")

def handle_response(resp):
    if resp is not None and isinstance(resp, Response):
        if resp.data_type == const.PDT_S_RETURN_DATA and resp.ret_len != 0:
            if resp.ret_len == 0:
                logging.info("Return data is empty")
                return
            
            ret_struct = msgpack.unpackb(resp.ret)
            
            if ret_struct['Code'] != 0:
                logging.info(ret_struct['Msg'])
                return
            
            print(ret_struct['Data'].decode("utf-8"))
            return

if __name__ == "__main__":
    main()