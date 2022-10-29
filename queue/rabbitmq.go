package queue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"git.ziniao.com/webscraper/go-gin-http/queue/consumer"
	"git.ziniao.com/webscraper/go-orm/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMqSt struct {
	Url        string `yaml:"url"`
	Durable    bool   `yaml:"durable"`
	AutoAck    bool   `yaml:"auto_ack"`
	AutoDelete bool   `yaml:"auto_delete"`
	QueueSt           //集成基础结构体
	state      map[string]*consumer.RabbitMqStateSt
	conn       *amqp.Connection
}

//获取一个消息对了的管道
func (s *RabbitMqSt) getState(topic string) *consumer.RabbitMqStateSt {
	if state, ok := s.state[topic]; ok && !state.Channel.IsClosed() {
		return state
	}
	s.Init() //完成初始化逻辑
	state := &consumer.RabbitMqStateSt{}
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
		body := s.format(data)
		pMsg = &amqp.Publishing{ContentType: "text/plain", Body: body}
	}
	s.l.Lock()
	defer func() {
		s.l.Unlock()
		if e := recover(); e != nil {
			err = errors.New(fmt.Sprintf("%+v", e))
		}
	}()
	ctx := context.Background()
	for i := 0; i < retryLimit; i++ {
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

// 初始化队列的链接处理逻辑 初始化的时候要锁定
func (s *RabbitMqSt) Init() (err error) {
	s.once.Do(func() {
		if s.topics == nil { //初始化一下
			s.topics = make(TopicConsumeSt)
		}
		s.state = make(map[string]*consumer.RabbitMqStateSt)
		s.conn, err = amqp.Dial(s.Url)
		if err != nil {
			log.Write(log.ERROR, "Failed connect ", err)
			panic(err)
		}
	})
	return err
}

// 关闭队列处理逻辑
func (s *RabbitMqSt) Close() {
	s.l.Lock()
	defer s.l.Unlock()
	if s.conn != nil { //关闭代理
		s.conn.Close()
		s.conn = nil
	}
	for topic, state := range s.state {
		state.Close()
		delete(s.state, topic)
	}
}

//启动服务的处理逻辑
func (s *RabbitMqSt) Start() (err error) {
	keepRunning := true
	s.Init() //完成初始化
	s.consumerCtx, s.consumerCancel = context.WithCancel(context.Background())
	//遍历注册consumer到消费组当中
	for topic, consumerWrapper := range s.topics {
		s.consumerStart(topic, &consumerWrapper)
	}
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	for keepRunning {
		select {
		case <-s.consumerCtx.Done():
			log.Write(-1, "terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			log.Write(-1, "terminating: via signal")
			keepRunning = false
		}
	}
	s.consumerCancel()
	s.consumerWg.Wait()
	s.Close() //执行退出了
	return nil
}

//开始一个执行消息的队列处理逻辑
func (s *RabbitMqSt) consumerStart(topic string, cWrapper *consumerWrapperSt) {
	cc := consumer.NewRabbitMqConsumer(cWrapper.conCurrency, s.AutoAck, topic, cWrapper.handle, s.Publish)
	s.consumerWg.Add(1)
	go func() {
		defer s.consumerWg.Done()
		for {
			state := s.getState(topic)
			if err := cc.ConsumeClaim(s.AutoAck, s.consumerCtx, state); err != nil {
				log.Write(log.ERROR, "Error from consumer:", err)
			} //独立线程
			log.Write(-1, "Consume 重新进入操作逻辑...", s.consumerCtx.Err())
			// check if context was cancelled, signaling that the consumer should stop
			if s.consumerCtx.Err() != nil {
				return
			}
		}
	}()
	log.Write(log.INFO, "rabbitmq consumer["+topic+"] up and running!...")
}
