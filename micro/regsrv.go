package micro

import (
	"os"
	"strings"
	
	"github.com/leicc520/go-gin-http"
	"github.com/leicc520/go-orm/log"
)

//不设置的话环境变量获取注册地址
func NewHttpMicroRegSrv(srv string) core.MicroClient {
	if len(srv) == 0 {
		srv = os.Getenv(core.DCSRV)
	}
	//非http请求的地址的情况
	if !strings.HasPrefix(srv,"http") {
		srv = "http://"+srv
	}
	disSrv := &HttpMicroRegSrv{regSrv: srv, jwtKey: os.Getenv(core.DCJWT),
		zHealth: map[string]HealthFunc{"http": defHttpZHealth}}
	log.Write(log.INFO, "register server:{"+srv+"} token:{"+os.Getenv(core.DCJWT)+"}")
	return core.MicroClient(disSrv)
}
