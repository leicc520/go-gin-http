package queue

import (
	"context"
	"encoding/json"
	"sync"
)

const (
	RetryLimit = 3
)

type ConsumerWrapperSt struct {
	ConCurrency int
	Handle      IFConsumer
}

// 定义一个基础队列queue->消费者的映射关系
type TopicConsumeSt map[string]ConsumerWrapperSt

type QueueSt struct {
	ConsumerCtx    context.Context    `yaml:"-"`
	ConsumerCancel context.CancelFunc `yaml:"-"`
	ConsumerWg     sync.WaitGroup     `yaml:"-"`
	Topics         TopicConsumeSt     `yaml:"-"` //注册的主题和消费者信息
	Once           sync.Once          `yaml:"-"` //执行初始化一次
	L              sync.Mutex         `yaml:"-"` //生产者发布消息的时候锁一下避免并发导致错误
}

// 获取队列的主题数据资料信息
func (s *QueueSt) GetTopics() []string {
	list, idx := make([]string, len(s.Topics)), 0
	for topic, _ := range s.Topics {
		list[idx] = topic
		idx++
	}
	return list
}

// 注册订阅队列和消费者绑定关闭
func (s *QueueSt) Register(topic string, consumer IFConsumer) {
	s.RegisterN(topic, -1, consumer)
}

// 注册订阅队列和消费者绑定关闭 允许设置并发开启线程数量
func (s *QueueSt) RegisterN(topic string, conCurrency int, consumer IFConsumer) {
	s.L.Lock()
	defer s.L.Unlock()
	if s.Topics == nil { //实例化
		s.Topics = make(TopicConsumeSt)
	}
	if s.Topics == nil {
		panic("队列未执行Init初始化操作")
	}
	s.Topics[topic] = ConsumerWrapperSt{Handle: consumer, ConCurrency: conCurrency}
}

// 格式化数据资料信息
func (s *QueueSt) Format(data interface{}) []byte {
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
	Close() //释放任务
	Publish(topic string, data interface{}) error
	Register(topic string, consumer IFConsumer)
	RegisterN(topic string, conCurrency int, consumer IFConsumer)
	Start(h func()) error //启动服务的处理逻辑
}
