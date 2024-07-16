import logging
import msgpack
import asyncio
import nmidsdk.model.const as const
from flask import Flask
from nmidsdk.client.client import *
# from concurrent.futures import Future

SERVER_HOST = "127.0.0.1"
SERVER_PORT = 6808

app = Flask(__name__)
@app.route('/test', methods=['GET'])        
def test():
    client = None
    result = ""
    # result_future = Future()
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)

    def err_handler(e):
        if e == const.RESTIMEOUT:
            logging.info("Time out here")
        else:
            logging.error(e)
        
        print("Client error here")

    def handle_response(resp):
        nonlocal result
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
                # result_future.set_result(ret_struct['Data'].decode("utf-8"))
                result = ret_struct['Data'].decode("utf-8")

    try:
        server_addr = (SERVER_HOST,SERVER_PORT)
        client = Client("tcp", server_addr)
        client_conn = client.start()
        if client_conn is None:
            logging.error("Failed to create client")
            return "Failed to create client"
    except Exception as e:
        logging.error(f"Error starting client: {e}")
        return f"Error starting client: {e}"

    client.err_handler = err_handler

    params_name1 = {"name": "nmid"}
    params1 = msgpack.packb(params_name1)

    client.do("ToUpper", bytearray(params1), handle_response)
    client.close()
    loop.close()

    # result = result_future.result()
    return result

if __name__ == '__main__':
    app.run(port=5981)