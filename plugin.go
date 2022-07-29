package core

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"plugin"
	"regexp"
	"strconv"
	"strings"

	"github.com/leicc520/go-orm/log"
)
/***********************************************************************
	统计定义微服务插件代码输入json字符串，输出对象+出错情况
	开启插件plugins的话需要 插件需要版本话管理，新的版本会覆盖旧版本
	插件命名规则go.micsrv.srv.xx.v20210701.so 服务明+版本日期
    新的版本上线之后调用重新加载，将会自动启用新的插件版本
 */
type microPlugins map[string]*plugin.Plugin
type PluginHandler func(string) (interface{}, error)
type pluginOptionSt struct {
	SoFile string
	Version int64
}
//定义全局的插件注册处理逻辑
var Gmicro = microPlugins{}
//扫码插件目录的so数据资料信息 插件命名规则go.micsrv.srv.uc.so 后缀.so
func (p *microPlugins) ScanAndLoadSo(pdir string) error  {
	dir, err := ioutil.ReadDir(pdir)
	if err != nil {
		log.Write(log.ERROR, "plugins dir scan error!", err)
		return errors.New("扫描加载so目录失败.")
	}
	regx, err := regexp.Compile("^(go\\.micsrv\\.srv\\.[a-z0-9]+)\\.v([\\d]+)\\.so$")
	if err != nil {
		return errors.New("扫描加载so正则模板编译失败.")
	}
	options := make(map[string]pluginOptionSt)
	for _, file := range dir {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".so") {
			continue
		}
		sliceStr := regx.FindStringSubmatch(file.Name())
		if sliceStr == nil || len(sliceStr) != 3 {
			continue
		}
		service := sliceStr[1]
		version, _ := strconv.ParseInt(sliceStr[2], 10, 64)
		sfile := filepath.Join(pdir, file.Name())
		//不存在或者已存在但服务版本比较低的情况 直接覆盖
		if sitem, ok := options[service]; !ok || (ok && sitem.Version < version) {
			options[service] = pluginOptionSt{SoFile: sfile, Version: version}
		} else {
			log.Write(log.INFO, "load so("+sfile+") service("+service+") 跳过低版本.")
		}
	}
	for service, items := range options {
		if pObj, err := plugin.Open(items.SoFile); err != nil {
			log.Write(log.ERROR, "file ("+items.SoFile+") service("+service+") load plugins error.", err)
		} else {//加载文件数据信息
			(*p)[service] = pObj
			log.Write(log.DEBUG, "load so("+items.SoFile+") service("+service+") success.")
		}
	}
	return nil
}

//扫码插件目录的so数据资料信息
func (p *microPlugins) Call(service, method, args string) (interface{}, error)  {
	defer func() {
		if err := recover(); err != nil {
			log.Write(log.ERROR, "plugins call recover.", err)
		}
	}()
	if _, ok := (*p)[service]; !ok {
		log.Write(log.ERROR, "404 request service("+service+") not found!")
		return nil, errors.New("404请求的服务不存在.")
	}
	handle, err := (*p)[service].Lookup(method)
	if err != nil {
		log.Write(log.ERROR, "404 request service method("+method+") not found!", err)
		return nil, errors.New("404请求的服务方法不存在.")
	}
	//强行转 然后直接执行业务代码
	datas, err := handle.(PluginHandler)(args)
	if err != nil {
		log.Write(log.ERROR, "500 request service("+service+") method("+method+") error!", err)
		return nil, errors.New("500请求服务出现错误.")
	}
	return datas, err
}
