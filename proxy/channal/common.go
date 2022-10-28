package channal

import (
	"github.com/go-redis/redis"
	"regexp"
	"sync"
)

const (
	EASY_GO_PROXY         = "http://zltiqu.pyhttp.taolop.com"
	SKY_GO_PROXY          = "http://api.tianqiip.com"
	PYNXX_GO_PROXY        = "http://tiqu.py.cn"
	IPIP_GO_PROXY         = "http://api.ipipgo.com"
	DAMAI_GO_PROXY        = "http://www.damaiip.com"
	SHENLONG_GO_PROXY     = "http://api.shenlongip.com"
	IDEA_GO_PROXY         = "http://api.proxy.ipidea.io"
	PROXY_SOCK5           = "tcp"
	PROXY_HTTPS           = "https"
	PROXY_HTTP            = "http"
	PROXY_CHANNEL_IPIPGO  = "ipipgo"
	PROXY_CHANNEL_EASYGO  = "easygo"
	PROXY_CHANNEL_SHENLGO = "shenlgo"
	PROXY_CHANNEL_IDEAGO  = "ideago"
	PROXY_CHANNEL_XXPYGO  = "xxpygo"
	PROXY_CHANNEL_DAMAIGO = "damaigo"
	PROXY_CHANNEL_SKYGO   = "skygo"

	PROXY_REDIS_PREFIX = "proxy@ip"
)

type IFProxy interface {
	GetParam() string
	SetIP(ip []string)
	SetParam(params string, pTime int64) IFProxy
	GetProxy(proto string) string
	CutProxy(proto string) string
}

var (
	proxyOnce                           = sync.Once{}
	distributedCache *redis.Client      = nil
	proxyDriver      map[string]IFProxy = nil
	regIpCheck, _                       = regexp.Compile(`[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}\.[\d]{1,3}:[\d]+`)
)

// 代理注册到注册器当中
func proxyRegister(name string, ifProxy IFProxy) {
	proxyOnce.Do(func() { //初始化逻辑
		proxyDriver = make(map[string]IFProxy)
	})
	proxyDriver[name] = ifProxy
}

// 获取代理IP数据资料信息
func GetProxy(name, params, proto string, expire int64) string {
	if proxyDriver == nil {
		return ""
	}
	if s, ok := proxyDriver[name]; ok {
		return s.SetParam(params, expire).GetProxy(proto)
	}
	return ""
}

// 强制切代理IP数据资料信息
func CutProxy(name, params, proto string, expire int64) string {
	if proxyDriver == nil {
		return ""
	}
	if s, ok := proxyDriver[name]; ok {
		return s.SetParam(params, expire).CutProxy(proto)
	}
	return ""
}

// 设置缓存策略数据信息
func SetRedis(rds *redis.Client) {
	distributedCache = rds
}

/******************************************************
  把系统支持的代理放写到这里，进行管理 需要加白明名单才能请求
*/
