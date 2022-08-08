package micro

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/leicc520/go-gin-http"
	"github.com/leicc520/go-orm/cache"
	"github.com/leicc520/go-orm/log"
)

/**************************************************************************
	基于http协议的简易服务发现处理逻辑 + 配置加载处理逻辑
 */
func InitMicroHttp() {
	core.SetRegSrv(func(srv string) core.MicroClient {
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
	})
}

type HealthFunc func(srv string) bool
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

type httpRegSevResponse struct {
	Code int64
	Srv string
}

type httpDiscoverSevResponse struct {
	Code int64 	  `json:"code"`
	Msg  string   `json:"msg"`
	Srvs []string `json:"srvs"`
}

type httpConfigSevResponse struct {
	Code int64 	`json:"code"`
	Msg  string `json:"msg"`
	Yaml string `json:"yaml"`
}

type HttpMicroRegSrv struct {
	regSrv  string
	jwtKey  string
	zHealth map[string]HealthFunc
}

//获取服务器的地址信息
func (a *HttpMicroRegSrv) GetRegSrv() string {
	return a.regSrv
}

//检测微服务端状态 --最多尝试三次
func (a *HttpMicroRegSrv) Health(nTry int, protoSt, srv string) bool {
	status := false //默认失败的情况
	if handle, ok := a.zHealth[protoSt]; ok && handle != nil {
		for i := 0; i < nTry; i++ {
			status = handle(srv)
			if status {//状态检测到的情况
				break
			}
		}
	}
	return status
}

//注册心跳处理事件
func (a *HttpMicroRegSrv) SetHealthFunc(protoSt string, handle HealthFunc) *HttpMicroRegSrv {
	a.zHealth[protoSt] = handle
	return a
}

//发起一个网络请求
func (a *HttpMicroRegSrv) _request(url string, body []byte, method string) (result []byte) {
	var sp *http.Response = nil
	defer func() {//补货异常的处理逻辑
		if sp != nil && sp.Body != nil {
			sp.Body.Close()
		}
		if r := recover(); r != nil {
			log.Write(log.ERROR, "request url ", url, "error", r)
			result = nil
		}
	}()
	log.Write(log.INFO, url, string(body), a.jwtKey)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		log.Write(log.ERROR, url, err, string(body))
		return nil
	}
	req.Header.Set("X-TOKEN", a.jwtKey)
	req.Header.Set("Content-Type", "application/json")
	client  := &http.Client{Timeout: time.Second*3}
	sp, err = client.Do(req)
	if err != nil || sp == nil || sp.StatusCode != http.StatusOK {
		log.Write(log.ERROR, url, err, string(body))
		return nil
	}
	if result, err = ioutil.ReadAll(sp.Body); err != nil {
		log.Write(log.ERROR, url, err, string(body))
		return nil
	} else {
		return result
	}
}

//提交申请接口注册处理逻辑 返回注册的服务地址
func (a *HttpMicroRegSrv) Register(name, srv, protoSt, version string) string {
	req := map[string]interface{}{"name":name,
		"srv":srv, "proto":protoSt, "version":version}
	body, _ := json.Marshal(req)
	data := a._request(a.regSrv+"/micsrv/register", body, "POST")
	if data == nil || len(data) < 3 {//请求返回异常直接panic
		panic(errors.New("Register microsrv{"+name+"} error"))
	}
	srvAddr := httpRegSevResponse{}
	if err := json.Unmarshal(data, &srvAddr); err != nil || srvAddr.Code != 0 {
		log.Write(log.FATAL, "Register",  err, srvAddr)
		panic(errors.New("Register microsrv{"+name+"} error"))
	}
	log.Write(-1, "register server{"+name+"} success")
	return srvAddr.Srv
}

//提交申请注销微服务的处理逻辑
func (a *HttpMicroRegSrv) UnRegister(protoSt, name, srv string)  {
	req  := map[string]string{"name":name, "proto":protoSt, "srv":srv}
	body, _ := json.Marshal(req)
	data := a._request(a.regSrv+"/micsrv/unregister", body, "POST")
	log.Write(log.INFO, "unregister server{"+name+"-->"+srv+"}-{"+string(data)+"} success")
}

//提交请求申请微服务发现逻辑
func (a *HttpMicroRegSrv) Discover(protoSt, name string) ([]string, error) {
	url := a.regSrv+"/micsrv/discover/"+protoSt+"/"+name
	data := a._request(url, nil, "GET")
	if data == nil {//服务异常的情况
		return nil, errors.New("发现服务异常,无法获得数据.")
	}
	log.Write(log.INFO, url, string(data))
	srvs := httpDiscoverSevResponse{}
	if err := json.Unmarshal(data, &srvs); err != nil || srvs.Code != 0 {
		return nil, err
	}
	return srvs.Srvs, nil
}

//加载配置数据资料信息
func (a *HttpMicroRegSrv) _config(name string) string {
	url := a.regSrv+"/micsrv/config/"+name
	data := a._request(url, nil, "GET")
	if data == nil || len(data) < 1 {//服务异常的情况
		return ""
	}
	log.Write(log.INFO, url, string(data))
	item := httpConfigSevResponse{}
	if err := json.Unmarshal(data, &item); err != nil || item.Code != 0 {
		return ""
	}
	return item.Yaml
}

//获取微服务配置管理 配置写文件缓存
func (a *HttpMicroRegSrv) Config(name string) string {
	cache := cache.NewFileCache("./cachedir", 1)
	if yaml := a._config(name); len(yaml) > 0 {
		cache.Set("config@"+name, yaml, 0)
		return yaml
	}
	item := cache.Get("config@"+name)
	if item != nil {//数据不为空的情况
		if yaml, ok := item.(string); ok && len(yaml) > 0 {
			return yaml
		}
	}
	log.Write(log.ERROR, "load Config {"+name+"} failed")
	panic("load Config {"+name+"} failed")
}

//提交请求申请微服务发现逻辑
func (a *HttpMicroRegSrv) Reload() error {
	data := a._request(a.regSrv+"/micsrv/reload", nil, "GET")
	if data == nil {//服务异常的情况
		return errors.New("重启服务异常,无法获得数据.")
	}
	log.Write(log.INFO, a.regSrv+"/micsrv/reload")
	log.Write(log.INFO, string(data))
	srvs := struct {
		Code int64 	`json:"code"`
		Msg  string `json:"msg"`
	}{}
	if err := json.Unmarshal(data, &srvs); err != nil || srvs.Code != 0 {
		return errors.New(srvs.Msg)
	}
	return nil
}