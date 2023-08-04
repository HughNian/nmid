package worker

import (
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func EtcdClient(addrs []string) *clientv3.Client {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   addrs,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		// handle error!
		fmt.Printf("connect to etcd failed, err:%v\n", err)
		return nil
	}

	return cli
}
