jaegerTracing 链路追踪接入处理逻辑
https://www.jianshu.com/p/b5cd7b07a24e

安装docker 仅供测试
docker run -d -p 6831:6831/udp -p 16686:16686 jaegertracing/all-in-one:latest

集成jaeger分布式链路跟踪处理逻辑，可以实现动态开启或者关闭链路跟踪

gin整合endless实现服务热重启
