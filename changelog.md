### 对服务进行优化
- 提升服务的并发，性能 √
- 对服务的启动和常驻进程的代码逻辑进行合理优化 √
- 对代码中高并发场景进行优化，该用锁的地方加上 √
- 增加限流操作，client,worker两处限流，codel和令牌桶限流 √

### 增加注册中心
- 现有的worker注册方式保持，即直接注册到nmid中。
- 增加注册中心服务，worker注册到注册中心服务，nmid从注册中心拉取worker。
- worker的标识增加，包含serviceid,servicename,funcname

### 增加worker_key概念
- 创建worker时可以生产worker_key也可以设置worker_key，使一组worker属于同一个worker_key下，这样对同worker_key下的worker组进行节点集群的处理，主要的有可以根据raft共识算法，选举产生leader worker,代表这worker组对外提供服务。
- 使用raft共识算法，但目前只用到选举产生leader的逻辑，暂无同步快照逻辑。