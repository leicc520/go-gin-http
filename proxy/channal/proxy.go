package channal

import (
	"errors"
	"strings"
	"time"

	"git.ziniao.com/webscraper/go-gin-http/locker"
	"git.ziniao.com/webscraper/go-orm/log"
)

type proxyHandleSt func(proto string, proxy IFProxy) error

type BaseProxySt struct {
	idx       int
	params    string
	name      string
	proxyTime time.Duration
	getProxy  proxyHandleSt
	ipList    []string //改成通过redis获取
}

// 检测是否含有IP
func (s *BaseProxySt) ContainsIP(str string) bool {
	return regIpCheck.MatchString(str)
}

// 初始化处理逻辑
func (s *BaseProxySt) init(name string, pHandle proxyHandleSt) {
	s.getProxy = pHandle
	s.proxyTime, s.name = time.Second*60, name
}

// 设置代理的情况逻辑
func (s *BaseProxySt) SetParam(params string, pTime int64) IFProxy {
	s.params = params
	s.proxyTime = time.Second * time.Duration(pTime)
	return s
}

// 获取请求参数
func (s *BaseProxySt) GetParam() string {
	return s.params
}

// 更新IP列表
func (s *BaseProxySt) SetIP(ip []string) {
	ipStr := strings.Join(ip, ",")
	if !s.ContainsIP(ipStr) { //提取IP报错的情况
		log.Write(-1, s.name, "提取IP返回错误...", ipStr)
		return
	}
	if distributedCache != nil { //最长1天的缓存
		distributedCache.Set(PROXY_REDIS_PREFIX+s.name, ipStr, s.proxyTime)
	}
	log.Write(log.INFO, "代理生成", s.name, ipStr)
	s.ipList = ip
}

// 更新代理的处理逻辑
func (s *BaseProxySt) callProxy(proto string) error {
	if distributedCache == nil {
		return errors.New("ip代理未设置管理redis,无法操作")
	}
	locker := locker.NewRedisLock(distributedCache, s.name+"v2")
	if !locker.Expire(s.proxyTime).Lock() { //获取锁失败的情况
		return nil
	}
	defer locker.UnLock() //设置解锁
	return s.getProxy(proto, s)
}

// 通过缓存获取IP数据信息
func (s *BaseProxySt) getIpList(proto string) []string {
	if distributedCache == nil { //最长1天的缓存
		return s.ipList
	}
	cmd := distributedCache.Get(PROXY_REDIS_PREFIX + s.name)
	if cmd != nil {
		ipStr := cmd.Val()
		if len(ipStr) > 0 {
			return strings.Split(ipStr, ",")
		}
	}
	//redis没有取到的话重新取一下数据
	s.callProxy(proto)
	return s.ipList
}

// 切换代理的处理逻辑
func (s *BaseProxySt) CutProxy(proto string) string {
	if err := s.callProxy(proto); err != nil {
		return ""
	}
	return s.GetProxy(proto)
}

// 获取代理IP 数据资料信息
func (s *BaseProxySt) GetProxy(proto string) string {
	ipList := s.getIpList(proto)
	if ipList == nil || len(ipList) < 1 {
		return ""
	}
	s.idx += 1
	s.idx = (s.idx + 1) % len(ipList)
	log.Write(-1, s.name, "切换代理", proto, ipList[s.idx])
	return strings.TrimSpace(ipList[s.idx])
}
