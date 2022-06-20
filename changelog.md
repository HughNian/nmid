### 对服务进行优化
- 提升服务的并发，性能 √
- 对服务的启动和常驻进程的代码逻辑进行合理优化 √
- 对代码中高并发场景进行优化，该用锁的地方加上 √
- 增加限流操作，client,worker两处限流，codel和令牌桶限流 √

### v2.0.1+建设
- 增加注册中心
- worker增加心跳机制
- 创建worker时可以设置worker_label，相同worker_label的worker为同一组
- worker的标识增加，包含worker_label，serviceid，servicename，funcname。worker可以抽象为service
- worker与client通过ipfs协议存在调用关系
- 同一组worker连接同一个nmid服务
- worker连接nmid服务时，同时注册到注册中心服务，由pkg中worker操作
- client在请求前需要初始化挂载相应worker服务，注册关系到注册中心
- client请求时校验client与worker的调用关系，然后从注册中心拉取worker清单，由pkg中client操作

ps: client与worker的关系在nmid的微服务中是相互的，因为client有可能也是worker服务，同样worker可能也是某个其他worker的消费者client。
