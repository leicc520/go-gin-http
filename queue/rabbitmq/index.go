package rabbitmq

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.ziniao.com/webscraper/go-gin-http/queue"
	"git.ziniao.com/webscraper/go-orm/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMqSt struct {
	Url           string `yaml:"url"`
	Durable       bool   `yaml:"durable"`
	AutoAck       bool   `yaml:"auto_ack"`
	AutoDelete    bool   `yaml:"auto_delete"`
	queue.QueueSt        //集成基础结构体
	state         map[string]*rabbitMqStateSt
	conn          *amqp.Connection
}

// 获取一个消息对了的管道
func (s *RabbitMqSt) getState(topic string) *rabbitMqStateSt {
	if state, ok := s.state[topic]; ok && !state.Channel.IsClosed() {
		return state
	}
	s.init() //完成初始化逻辑
	state := &rabbitMqStateSt{}
	if err := state.Init(topic, s.Durable, s.AutoDelete, s.conn); err != nil {
		return nil
	}
	s.state[topic] = state
	return state
}

// 往队列发送消息 允许直接传递消息对象指针
func (s *RabbitMqSt) Publish(topic string, data interface{}) (err error) {
	var ok = false
	var pMsg *amqp.Publishing = nil
	if pMsg, ok = data.(*amqp.Publishing); !ok {
		body := s.Format(data)
		pMsg = &amqp.Publishing{ContentType: "text/plain", Body: body}
	}
	s.L.Lock()
	defer s.L.Unlock()
	ctx := context.Background()
	for i := 0; i < queue.RetryLimit; i++ {
		state := s.getState(topic)
		if state == nil {
			err = errors.New("获取队列[" + topic + "]消息管道异常.")
			continue
		}
		err = state.Channel.PublishWithContext(ctx, "", topic, false, false, *pMsg)
		if err != nil { //失败的情况处理逻辑
			log.Write(log.ERROR, "Failed to publish a message", err)
			continue
		}
		break
	}
	return err
}

// 连接队列服务的处理逻辑
func (s *RabbitMqSt) _connect() (err error) {
	if s.conn != nil && !s.conn.IsClosed() {
		return //已经连接成功
	}
	s.conn, err = amqp.Dial(s.Url)
	if err != nil {
		log.Write(log.ERROR, "Failed connect ", err)
		return
	}
	return nil
}

// 初始化队列的链接处理逻辑 初始化的时候要锁定
func (s *RabbitMqSt) init() (err error) {
	s.Once.Do(func() {
		if s.Topics == nil { //初始化一下
			s.Topics = make(queue.TopicConsumeSt)
		}
		s.state = make(map[string]*rabbitMqStateSt)
		if err = s._connect(); err != nil {
			panic(err)
		}
	})
	//重连最多尝试3次
	for i := 0; i < queue.RetryLimit; i++ {
		if err = s._connect(); err != nil {
			continue
		} //重连的检测
		time.Sleep(time.Second * time.Duration(i+1))
		break
	}
	return err
}

// 关闭队列处理逻辑
func (s *RabbitMqSt) Close() {
	s.L.Lock()
	defer s.L.Unlock()
	if s.conn != nil { //关闭代理
		s.conn.Close()
		s.conn = nil
	}
	for topic, state := range s.state {
		state.Close()
		delete(s.state, topic)
	}
}

// 启动服务的处理逻辑
func (s *RabbitMqSt) Start() (err error) {
	keepRunning := true
	s.init() //完成初始化
	s.ConsumerCtx, s.ConsumerCancel = context.WithCancel(context.Background())
	//遍历注册consumer到消费组当中
	for topic, consumerWrapper := range s.Topics {
		s.consumerStart(topic, &consumerWrapper)
	}
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	for keepRunning {
		select {
		case <-s.ConsumerCtx.Done():
			log.Write(-1, "terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			log.Write(-1, "terminating: via signal")
			keepRunning = false
		}
	}
	s.ConsumerCancel()
	s.ConsumerWg.Wait()
	s.Close() //执行退出了
	return nil
}

// 开始一个执行消息的队列处理逻辑
func (s *RabbitMqSt) consumerStart(topic string, cWrapper *queue.ConsumerWrapperSt) {
	cc := NewRabbitMqConsumer(cWrapper.ConCurrency, s.AutoAck, topic, cWrapper.Handle, s.Publish)
	s.ConsumerWg.Add(1)
	go func() {
		defer s.ConsumerWg.Done()
		for {
			state := s.getState(topic)
			if err := cc.ConsumeClaim(s.AutoAck, s.ConsumerCtx, state); err != nil {
				log.Write(log.ERROR, "Error from consumer:", err)
			} //独立线程
			log.Write(-1, "Consume 重新进入操作逻辑...", s.ConsumerCtx.Err())
			// check if context was cancelled, signaling that the consumer should stop
			if s.ConsumerCtx.Err() != nil {
				return
			}
		}
	}()
	log.Write(log.INFO, "rabbitmq consumer["+topic+"] up and running!...")
}
