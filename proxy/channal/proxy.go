package channal

import (
	"strings"
	"sync"
	"time"

	"git.ziniao.com/webscraper/go-gin-http/locker"
	"git.ziniao.com/webscraper/go-orm/log"
)

type proxyHandleSt func(proto string, proxy IFProxy) error

type BaseProxySt struct {
	isInit    bool
	idx       int
	n         sync.Once
	l         sync.RWMutex
	params    string
	name      string
	cTime     int64
	proxyTime time.Duration
	getProxy  proxyHandleSt
	notify    chan bool
	ipList    []string //改成通过redis获取
}

// 初始化处理逻辑
func (s *BaseProxySt) init(name string, pHandle proxyHandleSt) {
	s.notify = make(chan bool)
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
	s.l.Lock()
	defer s.l.Unlock()
	ipStr := strings.Join(ip, ",")
	if distributedCache != nil { //最长1天的缓存
		distributedCache.Set(PROXY_REDIS_PREFIX+s.name, ipStr, s.proxyTime)
	}
	s.ipList = ip
	s.isInit = true
}

// 通过缓存获取IP数据信息
func (s *BaseProxySt) GetIpList() []string {
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
	return s.ipList
}

// 通过缓存获取IP数据信息
func (s *BaseProxySt) ipStr() string {
	cmd := distributedCache.Get(PROXY_REDIS_PREFIX + s.name)
	if cmd != nil {
		return cmd.Val()
	}
	return ""
}

// 切换代理的处理逻辑
func (s *BaseProxySt) CutProxy(proto string) string {
	s.notify <- true //通知强制更新
	time.Sleep(time.Second * 1)
	return s.GetProxy(proto)
}

// 检测是否重新开始接管生成ip的逻辑
func (s BaseProxySt) checkStart() bool {
	nTry := 0
	log.Write(-1, "已经被其他服务锁定,无法解锁", s.name)
	for {
		select {
		case <-time.After(s.proxyTime):
			if ipStr := s.ipStr(); len(ipStr) < 1 {
				nTry++
				log.Write(-1, "已经被其他服务锁定,解锁检测", s.name)
				if nTry >= 3 { //连续三次见到ip为空，说明更新ip的协程挂了
					return true
				}
			} else { //重置0
				nTry = 0
			}
		}
	}
	return false
}

// 只执行一次的业务处理逻辑
func (s *BaseProxySt) onceStart(proto string) {
	if distributedCache == nil {
		log.Write(-1, "未设置分布式缓存策略【SetRedis】....")
		panic("未设置分布式缓存策略....")
	}
	locker := locker.NewRedisLock(distributedCache, s.name)
	if !locker.Expire(-1).Lock() { //获取锁失败的情况
		if s.checkStart() {
			log.Write(-1, "自动检测释放代理锁", s.name)
			locker.UnLock()
			if !locker.Expire(-1).Lock() {
				return
			}
		} else {
			return
		}
	}
	defer locker.UnLock() //设置解锁
	//如果代理IP没有设置的情况逻辑
	if ip := s.GetIpList(); ip == nil || len(ip) < 1 {
		s.getProxy(proto, s) //首次初始化
	}
	for { //每3分钟中自动切换一下IP
		select {
		case <-s.notify:
			err := s.getProxy(proto, s)
			log.Write(-1, s.name, "紧急代理切换", err)
		case <-time.After(s.proxyTime):
			err := s.getProxy(proto, s)
			log.Write(-1, s.name, "定时代理切换", err)
		}
	}
}

// 获取代理IP 数据资料信息
func (s *BaseProxySt) GetProxy(proto string) string {
	s.n.Do(func() { //只要启动执行一次即可
		go s.onceStart(proto)
	})
	if !s.isInit { //没有初始化的情况
		time.Sleep(time.Second * 10)
	}
	s.l.RLock()
	defer s.l.RUnlock()
	ipList := s.GetIpList()
	if ipList == nil || len(ipList) < 1 {
		return ""
	}
	s.idx = (s.idx + 1) % len(ipList)
	log.Write(log.INFO, s.name, "切换代理", ipList[s.idx])
	return strings.TrimSpace(ipList[s.idx])
}
