package proxy

import (
	"time"

	"git.ziniao.com/webscraper/go-gin-http/proxy/channal"
	"git.ziniao.com/webscraper/go-orm/log"
)

const (
	PROXY_ERROR_LOCK_TIME  = time.Millisecond * 100
	PROXY_SYNC_REDIS_TIME  = time.Second * 30
	PROXY_SYNC_NOTIFY_TIME = time.Hour * 24
	PROXY_ERROR_LIMIT      = 30        //连续错误30次切换ip
	PROXY_DEFUALT_NAME     = "default" //默认代理监控
)

type ProxySt struct {
	Proxy      string `yaml:"proxy"  json:"proxy"`          //代理名称
	Url        string `yaml:"url"    json:"url"`            //代理请求地址
	Params     string `yaml:"params" json:"params"`         //附加参数
	ProxyTime  int64  `yaml:"proxy_time" json:"proxy_time"` //代理失效时间
	Status     int8   `yaml:"status" json:"status"`         //状态0-禁用 1-正常 2-锁定
	IsTcp      bool   `yaml:"-" json:"-"`                   //是否tcp代理
	Error      uint64 `yaml:"-"      json:"-"`              //请求失败的统计
	ProxyError uint64 `yaml:"-"      json:"-"`              //代理失效的统计
	Expire     int64  `yaml:"-" json:"-"`                   //锁定时间
}

// 自动切换代理处理逻辑
func (s *ProxySt) CutProxy(isCut bool) {
	if s.Status < 1 { //关闭了代理
		return
	}
	ipAddress := ""
	if isCut { //切代理，错误太多的时候
		ipAddress = channal.CutProxy(s.Proxy, s.Params, channal.PROXY_SOCK5, s.ProxyTime)
		if len(ipAddress) > 0 { //切换ip的情况
			s.ProxyError, s.Error = 0, 0
		}
	} else { //更新获取代理ip
		ipAddress = channal.GetProxy(s.Proxy, s.Params, channal.PROXY_SOCK5, s.ProxyTime)
	}
	if len(ipAddress) > 0 { //请求失败很多更好地址
		s.Url = "http://" + ipAddress
		s.Expire, s.Status, s.IsTcp = 0, 1, true
	}
	log.Write(-1, "自动切换IP:", ipAddress)
}
