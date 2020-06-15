## nmid

![logo](https://raw.githubusercontent.com/HughNian/nmid/master/logo/nmidlogo%EF%BC%8850x65%EF%BC%89.png)

nmid 意思为中场指挥官，足球场上的中场就是统领进攻防守的核心。咱们这里是服务程序的调度核心。

#### I/O的通信协议
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