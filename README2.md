<div align="center">
    <a href="http://www.niansong.top"><img src="https://raw.githubusercontent.com/HughNian/nmid-c/master/logo/nmid_c_logo.png" alt="nmid Logo" width="160"></a>
</div>

## nmid-c介绍

nmid-c是nmid微服务调度系统的客户端和工作端的C语言实现。同时可以编译成C动态库，在其他C程序中调用。nmid-php-ext作为php的扩展就是用了nmid-c项目的动态库。

1.client目录为客户端C语言源码目录，采用libev用作网络库，nmid的自有I/O通信协议，msgpack作为通信数据格式   

2.worker目录为工作端C语言源码目录，采用libev用作网络库，nmid的自有I/O通信协议，msgpack作为通信数据格式   

3.run目录为客户端，工作端C程序的可执行文件，可执行文件的编译用的是make  

4.build目录为客户端，工作端C程序的动态库目录，动态库的编译用的是cmake   


## 建议配置

```
cat /proc/version
Linux version 3.10.0-957.21.3.el7.x86_64 ...(centos7)

go version
go1.12.5 linux/amd64

gcc --version
gcc (GCC) 4.8.5 20150623 (Red Hat 4.8.5-36)

cmake --version
cmake version 3.11.4

```

## 需要的库

```
-lpthread, -lev, -lmsgpackc

```

## 编译安装步骤

```
https://github.com/HughNian/nmid-c.git

1.client
cd nmid-c/run/client
make

2.worker
cd nmid-c/run/worker
make

```

## 交流博客

http://www.niansong.top