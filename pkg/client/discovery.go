package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Consumer struct {
	EtcdAddrs []string
	Username  string
	Password  string
	EtcdCli   *clientv3.Client
	Workers   map[string][]string
}

func (c *Consumer) EtcdClient() *clientv3.Client {
	v3Config := clientv3.Config{
		Endpoints:   c.EtcdAddrs,
		DialTimeout: 5 * time.Second,
	}
	if c.Username != "" && c.Password != "" {
		v3Config.Username = c.Username
		v3Config.Password = c.Password
	}

	cli, err := clientv3.New(v3Config)

	if err != nil {
		// handle error!
		fmt.Printf("connect to etcd failed, err:%v\n", err)
		return nil
	}

	return cli
}

func (c *Consumer) EtcdWatch() {
	watchCh := c.EtcdCli.Watch(context.TODO(), model.EtcdBaseKey, clientv3.WithPrefix())
	go func() {
		for {
			<-watchCh

			c.Workers = c.GetAllWorkerIns(model.EtcdBaseKey)

			//do prometheus discovery count
			discoveryCount.Inc("etcd")
		}
	}()
}

func (c *Consumer) Discovery(funcName string) (client *Client) {
	if addrs, ok := c.Workers[funcName]; ok {
		index := utils.RandomInt(1, 10000) % len(addrs) //随机负载，todo使用更多的loadbalnce算法
		nmidAddr := addrs[index]

		client = NewClient("tcp", nmidAddr)
		return client
	}

	return
}

func (c *Consumer) GetAllWorkerIns(prefix string) map[string][]string {
	var allWorkerIns = make(map[string][]string)

	//use key prefix get the workerins
	resp, err := c.EtcdCli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return nil
	}

	for i := range resp.Kvs {
		if v := resp.Kvs[i].Value; v != nil {
			// fmt.Printf("%s-%s\n", resp.Kvs[i].Key, resp.Kvs[i].Value)

			var workerIns = make(map[string][]string)
			ret := json.Unmarshal(resp.Kvs[i].Value, &workerIns)
			if ret == nil {
				for key, val := range workerIns {
					allWorkerIns[key] = val
				}
			}
		}
	}

	return allWorkerIns
}
