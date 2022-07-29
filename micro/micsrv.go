package micro

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	jsonIter "github.com/json-iterator/go"
	"github.com/leicc520/go-orm"
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

type zHealthFunc func(srv string) bool
//默认http我服务检测
func defHttpZHealth(srv string) bool {
	client  := http.Client{Timeout: 3*time.Second}
	sp, err := client.Get("http://"+srv+"/healthz")
	defer func() {
		if sp != nil && sp.Body != nil {
			sp.Body.Close()
		}
	}()
	if err != nil || sp.StatusCode != http.StatusOK {
		log.Write(log.ERROR, sp, err)
		return false
	}
	return true
}

var gMicroRegSrv MicroClient = nil
var json = jsonIter.ConfigCompatibleWithStandardLibrary

//不设置的话环境变量获取注册地址
func NewMicroRegSrv(srv string) MicroClient {
	if len(srv) == 0 {
		srv = os.Getenv(DCSRV)
	}
	//只能生成一个全局即可
	if gMicroRegSrv == nil || gMicroRegSrv.GetRegSrv() != srv {
		//非http请求的地址的情况
		if !strings.HasPrefix(srv,"http") {
			srv = "http://"+srv
		}
		token := fmt.Sprintf("%x", md5.Sum([]byte(os.Getenv(DCJWT))))
		disSrv:= &HttpMicroRegSrv{regSrv: srv, jwtKey: token, zHealth: map[string]zHealthFunc{"http":defHttpZHealth}}
		log.Write(log.INFO, "register server:{"+srv+"} token:{"+token+"}")
		gMicroRegSrv = MicroClient(disSrv)
	}
	return gMicroRegSrv
}

//申请获取微服务注册的地址信息
func MicSrvServer(srv, protoSt, name string) string {
	rs    := NewMicroRegSrv(srv)
	cache := orm.GetMCache()
	//通过注册服务 获取数据资料信息 且记录到内存当中 失败的时候取
	srvs, err, ckey := []string{}, errors.New(""), protoSt+"@"+name
	if srvs, err = rs.Discover(protoSt, name); err != nil || len(srvs) < 1 {
		log.Write(log.ERROR, "服务发现地址获取异常{", name, "},通过cache检索")
		data := cache.Get(ckey)
		if data != nil {//数据不为空的情况
			srvs, _ = data.([]string)
		}
	} else {//数据获取成功的情况
		cache.Set(ckey, srvs, 0)
	}
	nidx := len(srvs) - 1
	if nidx > 0 {//大于2条记录做负载均衡
		nidx = int(time.Now().Unix()) % len(srvs)
	}
	if nidx >= 0 {
		return srvs[nidx]
	}
	return ""
}