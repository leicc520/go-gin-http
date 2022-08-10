package core

import (
	"errors"
	"time"
	
	"github.com/leicc520/go-orm"
	"github.com/leicc520/go-orm/cache"
	"github.com/leicc520/go-orm/log"
)

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
//服务发现的配置对象
var (
	RegSrv     MicroClient  = nil
	regCache   cache.Cacher = nil
)

//设置注册的服务发现协议http/grpc
func SetRegSrv(regSrvHandle MicroRegSrvHandle) {
	RegSrv     = regSrvHandle("") //默认获取服务信息
	regCache   = orm.GetMCache()
}

//申请获取微服务的地址信息
func MicroService(protoSt, name string) string {
	service, err, cKey := make([]string, 0), errors.New(""), protoSt+"@"+name
	if service, err = RegSrv.Discover(protoSt, name); err != nil || len(service) < 1 {
		log.Write(log.ERROR, "服务发现地址获取异常{", cKey, "},通过cache检索")
		if err = regCache.GetStruct(cKey, &service); err != nil {//数据不为空的情况
			return "" //地址获取失败的情况
		}
	} else {//数据获取成功覆盖设置的情况
		regCache.Set(cKey, service, 0)
	}
	nIndex := len(service) - 1
	if nIndex > 0 {//大于2条记录做负载均衡
		nIndex = int(time.Now().Unix()) % len(service)
	}
	if nIndex >= 0 {
		return service[nIndex]
	}
	return ""
}
