package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"git.ziniao.com/webscraper/go-orm/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

//获取rabbitmq状态
type RabbitMqStateSt struct {
	Version int //版本号管理，每次初始化+1，ack确认需要版本一直,否则丢弃
	Channel *amqp.Channel
	Queue   amqp.Queue
}

//关闭管道处理逻辑
func (s *RabbitMqStateSt) Close() {
	if s.Channel != nil { //关闭渠道
		s.Channel.Close()
		s.Channel = nil
	}
}

//初始化创建一个消息管道
func (s *RabbitMqStateSt) Init(topic string, durable, autoDelete bool, conn *amqp.Connection) (err error) {
	s.Channel, err = conn.Channel()
	if err != nil {
		log.Write(log.ERROR, "Failed to open a channel", err)
		return err
	}
	s.Queue, err = s.Channel.QueueDeclare(
		topic,      // name
		durable,    // durable
		autoDelete, // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		log.Write(log.ERROR, "Failed to declare a queue", err)
		return err
	}
	return
}

// 获取消费队列管道数据信息
func (s *RabbitMqStateSt) consumer(autoAck bool) (<-chan amqp.Delivery, error) {
	msgChan, err := s.Channel.Consume(
		s.Queue.Name, // queue
		"",           // consumer
		autoAck,      // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		log.Write(log.ERROR, s.Queue.Name, "Failed to register a consumer", s.Channel.IsClosed(), err)
	}
	return msgChan, err
}

// 消费者结构数据信息
type RabbitMqConsumerSt struct {
	state *RabbitMqStateSt
	ctx   context.Context
	l     sync.Mutex
	consumerSt
}

// 创建一个消费者处理逻辑
func NewRabbitMqConsumer(conCurrency int, autoAck bool, topic string, h IFConsumer, p pMessage) *RabbitMqConsumerSt {
	c := &RabbitMqConsumerSt{}
	c.init(conCurrency, autoAck, topic, h, p)
	return c
}

// 确认队列的处理逻辑 释放
func (c *RabbitMqConsumerSt) Ack(deliveryTag uint64, err error) error {
	c.l.Lock()
	defer c.l.Unlock()
	//如果是非自动确认的话 需要手动确认 需要版本一直
	if !c.autoAck {
		if err == nil {
			err = c.state.Channel.Ack(deliveryTag, false)
		} else {
			err = c.state.Channel.Reject(deliveryTag, true)
		}
		if err != nil {
			log.Write(-1, "队列ACK确认异常", err)
		}
	}
	return nil
}

//同步的消费处理逻辑
func (c *RabbitMqConsumerSt) syncConsumer(dMsg amqp.Delivery) {
	sTime := time.Now()
	err := c.consumer.Accept(c.topic, dMsg.Body) //业务只需关注数据即可
	log.Write(log.INFO, c.topic, "任务执行时长:", time.Since(sTime))
	c.Ack(dMsg.DeliveryTag, err) //处理完结确认
}

// 启动异步执行处理逻辑
func (c *RabbitMqConsumerSt) asyncConsumer(dMsg amqp.Delivery) {
	c.goConChan <- 1
	go func(dlMsg amqp.Delivery) {
		defer func() { //结束释放并发位置
			<-c.goConChan
			if e := recover(); e != nil {
				log.Write(-1, c.topic, "queue handle panic", e)
			}
		}()
		sTime := time.Now()
		err := c.consumer.Accept(c.topic, dlMsg.Body) //业务只需关注数据即可
		log.Write(log.INFO, c.topic, "任务执行时长:", time.Since(sTime))
		c.Ack(dlMsg.DeliveryTag, err) //处理完结确认
	}(dMsg)
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *RabbitMqConsumerSt) ConsumeClaim(autoAck bool, ctx context.Context, state *RabbitMqStateSt) (err error) {
	var msgChan <-chan amqp.Delivery = nil
	msgChan, err = state.consumer(autoAck)
	if err != nil {
		return err
	}
	c.state, c.ctx = state, ctx //复制到对象内部
	defer func() {
		if e := recover(); e != nil {
			log.Write(-1, c.topic, "ConsumeClaim 结束异常", e)
			err = errors.New(fmt.Sprintf("%+v", e))
		}
	}()
	for {
		select {
		case message, isOk := <-msgChan:
			log.Write(log.INFO, c.topic, " Received a message: ", string(message.Body), isOk, err)
			if !isOk { //有的时候会异常关闭的情况逻辑
				log.Write(-1, c.topic, "queue closed", c.state.Channel.IsClosed(), message)
				if c.state.Channel.IsClosed() {
					err = errors.New(c.topic + " queue closed")
					return err
				}
				continue
			}
			if c.conCurrency <= 1 { //但协程的处理逻辑
				c.syncConsumer(message)
			} else {
				c.asyncConsumer(message)
			}
		case <-c.ctx.Done():
			log.Write(-1, "RabbitMQ ConsumeClaim Done退出逻辑")
			return nil
		}
	}
	return nil
}
