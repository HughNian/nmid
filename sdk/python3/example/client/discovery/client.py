import logging
from flask import Flask
from nmidsdk.client.discovery import *

SERVER_HOST = "127.0.0.1"
SERVER_PORT = 6808

dishost = "192.168.10.195"
disport = 2379
disusername = "root"
dispassword = "123456"

def get_client():
    client = None
    try:
        server_addr = (SERVER_HOST,SERVER_PORT)
        client = Client("tcp", server_addr)
        client_conn = client.start()
        if client_conn is None:
            logging.error("Failed to create client")
    except Exception as e:
        logging.error(f"Error starting client: {e}")
    
    return client

def discovery(func_name: str):
    client = consumer.discovery(func_name)
    if client is not None:
        client_conn = client.start()
        if client_conn is None:
            logging.error("Failed to create client")
    else:
        client = get_client()

    return client

app = Flask(__name__)
@app.route('/test', methods=['GET'])
def test():
    func_name = "ToUpper"

    client = discovery(func_name)
    client.close()

    return "result" 

if __name__ == '__main__':
    consumer = Consumer(dishost, disport, disusername, dispassword)
    consumer.etcd_client()
    consumer.etcd_watch()

    app.run(port=5981)