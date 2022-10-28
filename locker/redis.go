package locker

import (
	"github.com/go-redis/redis"
	"time"
)

type RedisLockerSt struct {
	rds  *redis.Client
	name string
	exp  time.Duration
}

// 创建一个锁对象
func NewRedisLock(rds *redis.Client, name string) *RedisLockerSt {
	return &RedisLockerSt{rds: rds, name: name, exp: lockerExpire}
}

// 设置过期时间的处理逻辑
func (s *RedisLockerSt) Expire(exp time.Duration) *RedisLockerSt {
	s.exp = exp //设置过期时间
	return s
}

// 分布式锁，避免多实例执行
func (s *RedisLockerSt) Lock() bool {
	cKey := lockerPrefix + s.name
	cmd := s.rds.Incr(cKey)
	if cmd == nil || cmd.Err() != nil || cmd.Val() > 1 {
		return false //已经有脚本执行了
	}
	if s.exp > 0 { //设置过期时间
		s.rds.Expire(cKey, s.exp)
	}
	return true
}

// 是否锁逻辑
func (s *RedisLockerSt) UnLock() {
	s.rds.Del(lockerPrefix + s.name)
}
