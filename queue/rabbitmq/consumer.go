package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"git.ziniao.com/webscraper/go-gin-http/queue"
	"git.ziniao.com/webscraper/go-orm/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

// 消费者结构数据信息
type RabbitMqConsumerSt struct {
	state *rabbitMqStateSt
	ctx   context.Context
	l     sync.Mutex
	queue.ConsumerSt
}

// 创建一个消费者处理逻辑
func NewRabbitMqConsumer(conCurrency int, autoAck bool, topic string, h queue.IFConsumer, p queue.PushMessage) *RabbitMqConsumerSt {
	c := &RabbitMqConsumerSt{}
	c.Init(conCurrency, autoAck, topic, h, p)
	return c
}

// 确认队列的处理逻辑 释放
func (c *RabbitMqConsumerSt) reset(deliveryTag uint64, isOk bool) (err error) {
	c.l.Lock()
	defer c.l.Unlock()
	//如果是非自动确认的话 需要手动确认 需要版本一直
	if !c.AutoAck {
		if isOk {
			err = c.state.Channel.Ack(deliveryTag, false)
		} else {
			err = c.state.Channel.Reject(deliveryTag, true)
		}
		if err != nil {
			log.Write(-1, "队列ACK确认异常", err)
		}
	}
	return
}

// 同步的消费处理逻辑
func (c *RabbitMqConsumerSt) syncConsumer(dMsg amqp.Delivery) {
	sTime := time.Now()
	isOk := c.Consumer.Accept(c.Topic, dMsg.Body) //业务只需关注数据即可
	log.Write(log.INFO, c.Topic, "任务执行时长:", time.Since(sTime))
	c.reset(dMsg.DeliveryTag, isOk) //处理完结确认
}

// 启动异步执行处理逻辑
func (c *RabbitMqConsumerSt) asyncConsumer(dMsg amqp.Delivery) {
	c.GoConChan <- 1
	go func(dlMsg amqp.Delivery) {
		defer func() { //结束释放并发位置
			<-c.GoConChan
			if e := recover(); e != nil {
				log.Write(-1, c.Topic, "queue handle panic", e)
			}
		}()
		sTime := time.Now()
		isOk := c.Consumer.Accept(c.Topic, dlMsg.Body) //业务只需关注数据即可
		log.Write(log.INFO, c.Topic, "任务执行时长:", time.Since(sTime))
		c.reset(dlMsg.DeliveryTag, isOk) //处理完结确认
	}(dMsg)
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *RabbitMqConsumerSt) ConsumeClaim(autoAck bool, ctx context.Context, state *rabbitMqStateSt) (err error) {
	var msgChan <-chan amqp.Delivery = nil
	msgChan, err = state.consumer(autoAck)
	if err != nil {
		return err
	}
	c.state, c.ctx = state, ctx //复制到对象内部
	defer func() {
		if e := recover(); e != nil {
			log.Write(-1, c.Topic, "ConsumeClaim 结束异常", e)
			err = errors.New(fmt.Sprintf("%+v", e))
		}
	}()
	for {
		select {
		case message, isOk := <-msgChan:
			log.Write(log.INFO, c.Topic, " Received a message: ", string(message.Body), isOk, err)
			if !isOk { //有的时候会异常关闭的情况逻辑
				log.Write(-1, c.Topic, "queue closed", c.state.Channel.IsClosed(), message)
				if c.state.Channel.IsClosed() {
					err = errors.New(c.Topic + " queue closed")
					return err
				}
				continue
			}
			if c.ConCurrency <= 1 { //但协程的处理逻辑
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
