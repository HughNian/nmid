package worker

import (
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdConfig struct {
	Addrs    []string
	Username string
	Password string
}

func EtcdClient(config EtcdConfig) *clientv3.Client {
	v3Config := clientv3.Config{
		Endpoints:   config.Addrs,
		DialTimeout: 5 * time.Second,
	}
	if config.Username != "" && config.Password != "" {
		v3Config.Username = config.Username
		v3Config.Password = config.Password
	}

	cli, err := clientv3.New(v3Config)

	if err != nil {
		// handle error!
		fmt.Printf("connect to etcd failed, err:%v\n", err)
		return nil
	}

	return cli
}
