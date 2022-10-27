package proxy

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/leicc520/go-orm/log"
)

//记录日志的状态 统计记录
type logStateSt struct {
	ProxyIdx 	int
	Proxy       string
	Host 		string
	Status 		int
}

//redis数值统计处理逻辑
type RedisStateSt struct {
	logChan chan string
	proxy   []string
	rds *redis.Client
}

//格式化成字符串
func (s logStateSt) toString() string {
	return fmt.Sprintf("%d;%s;%d;%s", s.ProxyIdx, s.Host, s.Status, s.Proxy)
}

//格式化状态数据资料信息
func logStateBuilder(logStr string) *logStateSt {
	arrStr := strings.Split(logStr, ";")
	if len(arrStr) != 4 {
		log.Write(log.ERROR, logStr, "代理监控数据异常...")
		return nil
	}
	proxyIdx, _ := strconv.ParseInt(arrStr[0], 10, 64)
	status, _   := strconv.ParseInt(arrStr[2], 10, 64)
	return &logStateSt{ProxyIdx:int(proxyIdx),
		Proxy: arrStr[3], Host: arrStr[1], Status: int(status)}
}

//统计监控数据资料信息
func NewRedisStateSt(rds *redis.Client) *RedisStateSt {
	logChan := make(chan string, 1024*10)
	ss := &RedisStateSt{rds: rds, logChan: logChan}
	go ss.goAsyncNotify() //开启异步执行队列 持久化数据到redis当中
	return ss
}

//返回内容消息管道处理逻辑
func (s *RedisStateSt) Chan() chan string {
	return s.logChan
}

//设置代理是否已经存在了
func (s *RedisStateSt) isExistsProxy(name string) bool {
	for _, eHost := range s.proxy {
		if eHost == name {//已经存在了
			return true
		}
	}
	return false
}

//添加代理到统计监控当中
func (s *RedisStateSt) AddProxy(proxy []string) {
	for _, host := range proxy {
		if !s.isExistsProxy(host) {
			s.proxy = append(s.proxy, host)
		}
	}
}

//当天剩余的时间处理逻辑
func (s *RedisStateSt) dayDuration() time.Duration {
	n := time.Now()
	l := time.Date(n.Year(), n.Month(), n.Day(), 23, 59, 59, 0, n.Location())
	t := l.Sub(n)
	return t
}

//异步任务通知，格式化存储到数据库
func (s *RedisStateSt) goAsyncNotify() {
	state      := make(map[string]int)
	syncChan   := time.After(PROXY_SYNC_REDIS_TIME)
	notifyChan := time.After(s.dayDuration())
	for {
		//接收请求处理逻辑
		select {
		case logStr, ok := <-s.logChan:
			if !ok {//句柄广告异常关闭了退出
				log.Write(-1, "async proxy monitor exit!")
				return
			}
			if _, ok = state[logStr]; !ok {
				state[logStr]  = 1
			} else {
				state[logStr] += 1
			}
			//数据存储的比较多 也做一次同步
			if len(state) > 256 {
				s.syncRedis(state)
			}
			log.Write(log.INFO, "完成代理状态收集...")
		case <-syncChan:
			s.syncRedis(state) //做一个定期同步处理逻辑
			syncChan = time.After(PROXY_SYNC_REDIS_TIME)
		case <-notifyChan:
			s.syncReset() //将redis数据清理并生产汇总报表
			notifyChan = time.After(PROXY_SYNC_NOTIFY_TIME)
		}
	}
}

//每日做一个重置处理逻辑
func (s *RedisStateSt) syncReset() {
	if s.rds == nil {
		return
	}
	for _, host := range s.proxy {
		ckey, state := s.state(host)
		s.rds.Del(ckey) //删除key信息
		if state != nil && state["proxy"] != host {
			continue
		}
		str := formatNotify(state)
		//todo 发送钉钉通知处理逻辑
		log.Write(log.DEBUG, host, str)
	}
}

//获取统计的数值状态信息
func (s *RedisStateSt) state(host string) (string, map[string]string) {
	ckey  := redisStatisticKey(host)
	cmd   := s.rds.HGetAll(ckey)
	state := cmd.Val()
	return ckey, state
}

//统计格式化统计数据资料信息
func formatNotify(state map[string]string) string {
	success, _ := strconv.ParseInt(state["success"], 10, 64)
	if success < 1 {
		success += 1
	}
	regCmp, _  := regexp.Compile(":[\\d]+$")
	failed, _  := strconv.ParseInt(state["failed"], 10, 64)
	ratio  := fmt.Sprintf("%.6f", float64(success) / float64(success + failed) * 100.00)
	strBuf := strings.Builder{}
	strBuf.WriteString("代理服务:"+state["proxy"]+"\r\n")
	strBuf.WriteString("状态200请求数:"+state["success"]+"\r\n")
	strBuf.WriteString("状态非200请求数:"+state["failed"]+"\r\n")
	strBuf.WriteString("计算成功率:"+ratio+"\r\n")
	strBuf.WriteString("请求失败明细:\r\n")
	for keyStr, val := range state {
		if ok := regCmp.MatchString(keyStr); ok {
			strBuf.WriteString("\t-"+keyStr+" 累计数:"+val+"\r\n")
		}
	}
	return strBuf.String()
}

//获取redis数据资料信息
func redisStatisticKey(proxy string) string {
	return "proxy@"+proxy
}

//将数据迁移到redis当中的处理逻辑
func (s *RedisStateSt) syncRedis(state map[string]int) {
	if s.rds == nil {//数据为空的情况
		return
	}
	for logStr, nSize := range state {
		logState := logStateBuilder(logStr)
		//丢弃异常的数据 数据处理逻辑 失败的情况
		if logState == nil {
			continue
		}
		//统计代理异常情况数据资料信息
		ckey, field := redisStatisticKey(logState.Proxy), "success"
		if logState.Status != http.StatusOK {
			field    = "failed"
		}
		s.rds.HSetNX(ckey, "proxy", logState.Proxy)
		s.rds.HIncrBy(ckey, field, int64(nSize))
		if logState.Status != http.StatusOK {//记录失败的域名明细
			field = fmt.Sprintf("%s:%d", logState.Host, logState.Status)
			s.rds.HIncrBy(ckey, field, int64(nSize))
		}
		delete(state, logStr)
	}
}