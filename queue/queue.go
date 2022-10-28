package queue

import (
	"context"
	"encoding/json"
	"sync"
	
	"git.ziniao.com/webscraper/go-gin-http/queue/consumer"
)

const (
	retryLimit = 3
)

type consumerWrapperSt struct {
	conCurrency int
	handle      consumer.IFConsumer
}

// 定义一个基础队列queue->消费者的映射关系
type TopicConsumeSt map[string]consumerWrapperSt

type QueueSt struct {
	consumerCtx    context.Context      `yaml:"-"`
	consumerCancel context.CancelFunc   `yaml:"-"`
	consumerWg     sync.WaitGroup       `yaml:"-"`
	topics TopicConsumeSt `yaml:"-"` //注册的主题和消费者信息
	once   sync.Once      `yaml:"-"` //执行初始化一次
	l      sync.Mutex     `yaml:"-"` //生产者发布消息的时候锁一下避免并发导致错误
}

// 获取队列的主题数据资料信息
func (s *QueueSt) Topics() []string {
	list, idx := make([]string, len(s.topics)), 0
	for topic, _ := range s.topics {
		list[idx] = topic
		idx++
	}
	return list
}

// 注册订阅队列和消费者绑定关闭
func (s *QueueSt) Register(topic string, consumer consumer.IFConsumer) {
	s.RegisterN(topic, -1, consumer)
}

// 注册订阅队列和消费者绑定关闭 允许设置并发开启线程数量
func (s *QueueSt) RegisterN(topic string, conCurrency int, consumer consumer.IFConsumer) {
	if s.topics == nil {
		panic("队列未执行Init初始化操作")
	}
	s.l.Lock()
	defer s.l.Unlock()
	s.topics[topic] = consumerWrapperSt{handle: consumer, conCurrency: conCurrency}
}

// 格式化数据资料信息
func (s *QueueSt) format(data interface{}) []byte {
	//字符串直接转字节数组即可
	var body []byte = nil
	if str, ok := data.(string); ok {
		body = []byte(str)
	} else if body, ok = data.([]byte); !ok {
		body, _ = json.Marshal(data)
	}
	return body
}

// 定义任务队列的处理逻辑
type IFQueue interface {
	Init() (err error)
	Publish(topic string, data interface{}) error
	Register(topic string, consumer consumer.IFConsumer)
	RegisterN(topic string, conCurrency int, consumer consumer.IFConsumer)
	Start() error //启动服务的处理逻辑
}
