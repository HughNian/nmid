import json
import random
import etcd3
import nmidsdk.model.const as const
from threading import Thread
from nmidsdk.client.client import Client

class Consumer:
    def __init__(self, etcd_host, etcd_port, username="", password=""):
        self.etcd_host = etcd_host
        self.etcd_port = etcd_port
        self.username = username
        self.password = password
        self.etcd_cli = None
        self.workers = {}

    def etcd_client(self):
        self.etcd_cli = etcd3.client(host="192.168.10.195", port=2379, user="root", password="123456", cert_key=None, cert_cert=None)
        return self.etcd_cli
    
    def etcd_watch(self):
        if self.etcd_cli is not None:
            watch = self.etcd_cli.watch_prefix(const.EtcdBaseKey)
            Thread(target=self._watch_loop, args=(watch,)).start()

    def _watch_loop(self, watch):
        for evnet in watch:
            self.workers = self.get_all_worker_ins(const.EtcdBaseKey)

    def get_all_worker_ins(self, prefix):
        all_worker_ins = {}
        try:
            resp = self.etcd_cli.get_prefix(prefix)
            for kv in resp:
                worker_ins = json.loads(kv.value.decode())
                all_worker_ins.update(worker_ins)
        except Exception as e:
            print(f"Error getting data from etcd: {e}")
            return None
        return all_worker_ins
    
    def discovery(self, func_name):
        if addrs := self.workers.get(func_name):
            index = random.randint(1, 10000) % len(addrs)
            nmid_addr = addrs[index]
            parts = nmid_addr.split(':')
            SERVER_HOST, SERVER_PORT = parts[0], int(parts[1])
            return Client("tcp", (SERVER_HOST,SERVER_PORT))
        
        return None