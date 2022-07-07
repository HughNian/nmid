### 对服务进行优化
- 提升服务的并发，性能 √
- 对服务的启动和常驻进程的代码逻辑进行合理优化 √
- 对代码中高并发场景进行优化，该用锁的地方加上 √
- 增加限流操作，client,worker两处限流，codel和令牌桶限流 √

### v2.0.1+建设
- 增加http的处理，使worker可以同时用tcp以及http进行访问 √
- nmid可以成为sidecar服务，增加支持service，service为http服务。
- 增加注册中心
- worker增加心跳机制
- 创建worker时可以设置worker_label，相同worker_label的worker为同一组，同一组worker连接同一个nmid服务。
- service与client通过ipfs协议存在调用关系
- client在请求前需要初始化挂载相应worker服务，注册关系到注册中心
- client请求时校验client与worker的调用关系，然后从注册中心拉取worker清单，由pkg中client操作

### v3.0.0+建设
- nmid使用etcd进行集群支持
- nmid的终端工具nmidctl，类似etcdctl
- nmid使用nmidctl进行流量和服务编排
- nmid支持k8s的IngressController

### ps 
- client与worker的关系在nmid的微服务中是相互的，因为client有可能也是worker服务，同样worker可能也是某个其他worker的client。
- nmid中的worker相当于无服务的function，就像faas。nmid中的service则为独立服务，nmid充当service的sidecar。
