<div align="center">
    <a href="http://www.niansong.top"><img src="https://raw.githubusercontent.com/HughNian/nmid/master/logo/nmidlogo.png" alt="nmid Logo" width="160"></a>
</div>

## nmid介绍

nmid意思为中场指挥官，足球场上的中场就是统领进攻防守的核心。咱们这里是服务程序的调度核心。是微服务调度系统。

1.server目录为nmid微服务调度服务端go实现，采用协程以及管道的异步通信，带有连接池。   

2.worker目录为nmid的工作端go实现，目前也有c语言实现，以及php扩展实现，可以实现golang, php, c等作为工作端，从而实现跨语言平台提供功能服务，目前在另外一个项目。            

3.client目录为nmid的客户端go实现，目前也有c语言实现，以及php扩展实现，可以实现golang, php, c等作为客户端，从而实现跨语言平台调用功能服务，目前在另外一个项目。   

4.run目录为demo运行目录。为go实现的客户端示例，调度服务端示例，客户端示例。目前调度服务端只有golang的实现。  

## I/O的通信协议

- 包结构   

    1.包头：链接类型[uint32/4字节]+数据类型[uint32/4字节]+包体长度[uint32/4字节]   
    
        连接类型：0初始，1服务端server，2工作端worker，3客户端client。    
        数据类型: server数据请求，server数据返回，worker数据请求...。    
        包体长度：具体返回包体数据内容的总长度。
    
    2.包体：  
        
        (1)client => sever: 客户端请求服务端  
        包体长度 = UINT32_SIZE + HandleLen + UINT32_SIZE + ParamsLen   
                  方法名长度值空间+方法名长度空间+msgpack后参数长度值空间+msgpack后参数长度空间
                  
        包体包含 = 方法长度值+方法名称+msgpack后参数长度值+msgpack后的参数值   
        
        client请求参数数据：参数都为字符串数组，入参为  
        []string{"order_sn:MBO993889253", "order_type:4"}，xx:xxx形式，以:分隔
        类似key:value。
        
        
        
        (2)server => worker: 服务端请求工作端
        包体长度 = UINT32_SIZE + HandleLen + UINT32_SIZE + ParamsLen   
                  方法名长度值空间+方法名长度空间+msgpack后参数长度值空间+msgpack后参数长度空间
                          
        包体包含 = 方法长度值+msgpack后参数长度值+方法名称+msgpack后的参数值    
        
        server请求参数数据：参数都为字符串数组，入参为  
        []string{"order_sn:MBO993889253", "order_type:4"}，xx:xxx形式，以:分隔
        类似key:value。可以理解为server做了client的透传。    
        
        
        
        (3)worker => server: 工作端返回数据服务端   
        包体长度 = UINT32_SIZE + HandleLen + UINT32_SIZE + ParamsLen + UINT32_SIZE + RetLen   
                  方法名长度值空间+方法名长度空间+msgpack后参数长度值空间+msgpack后参数长度空间+msgpack后结果长度值空间+msgpack后结果长度空间
                                  
        包体包含 = 方法长度值+方法名称+msgpack后参数长度值+msgpack后的参数值+msgpack后结果长度值+msgpack后结果值   
        
        worker返回结果数据：返回数据为统一格式结构体
        type RetStruct struct {
            Code int
            Msg  string
            Data []byte
        }       
        
        
        
        (4)server => client: 服务端返回数据客户端
        包体长度 = UINT32_SIZE + HandleLen + UINT32_SIZE + ParamsLen + UINT32_SIZE + RetLen   
        方法名长度值空间+方法名长度空间+msgpack后参数长度值空间+msgpack后参数长度空间+msgpack后结果长度值空间+msgpack后结果长度空间
                                  
        包体包含 = 方法长度值+msgpack后参数长度值+方法名称+msgpack后的参数值+msgpack后结果长度值+msgpack后结果值   
        
        worker返回结果数据：返回数据为统一格式结构体
        type RetStruct struct {
            Code int
            Msg  string
            Data []byte
        }
        可以理解为server做了worker的透传。