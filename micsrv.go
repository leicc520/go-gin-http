package core

//留下延展空间，未来可能会使用grpc协议做服务发现
type MicroClient interface {
	Health(nTry int, protoSt, srv string) bool
	Register(name, srv, protoSt, version string) string
	UnRegister(protoSt, name, srv string)
	Discover(protoSt, name string) ([]string, error)
	Config(name string) string
	GetRegSrv() string
	Reload() error
}

//设置注册函数处理逻辑
type MicroRegSrvHandle func(srv string) MicroClient
