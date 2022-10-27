package queue

import (
	"errors"
	"github.com/Shopify/sarama"
	"github.com/leicc520/go-orm/log"
	"time"
)

// 更新数据资料信息
type pushMessageHandle func(topic string, data interface{}) (err error)

// 消费者结构数据信息
type consumerSt struct {
	handle      QueueCB
	push        pushMessageHandle
	conCurrency int
	goConChan   chan int8
	autoAck     bool
	ready       chan bool
}

// 创建一个消费者处理逻辑
func newConsumer(conCurrency int, autoAck bool, h QueueCB, pushHandle pushMessageHandle) *consumerSt {
	var goConChan chan int8 = nil
	if conCurrency > 1 {
		goConChan = make(chan int8, conCurrency)
	}
	return &consumerSt{
		handle:    h,
		goConChan: goConChan,
		autoAck:   autoAck,
		push:      pushHandle,
		ready:     make(chan bool),
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *consumerSt) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(c.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *consumerSt) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// 重试机制的出来逻辑
func (c *consumerSt) reset(message *sarama.ConsumerMessage) {
	var err = errors.New("重试调用未执行")
	if c.push != nil { //更新数据信息
		err = c.push(message.Topic, message.Value)
	}
	log.Write(log.INFO, "Kafka重新重试机制", err)
}

// 异步消费的处理逻辑
func (c *consumerSt) asyncConsumer(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	var err error = nil
	c.goConChan <- 1
	go func(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
		defer func() {
			<-c.goConChan
			if e := recover(); e != nil {
				log.Write(-1, message.Topic, "queue handle panic", e)
			}
		}()
		log.Write(log.INFO, message.Topic, " Received a message: ", string(message.Value))
		sTime := time.Now()
		if err = c.handle(message.Value); err != nil { //处理成功标记完成 否则继续处理
			c.reset(message) //重试逻辑-再次入队列一次
		}
		session.MarkMessage(message, "")
		if !c.autoAck { //手动确认的情况逻辑
			session.Commit()
		}
		log.Write(log.INFO, message.Topic, "任务执行时长:", time.Since(sTime))
	}(session, message)
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *consumerSt) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	var err error = nil
	for {
		select {
		case message := <-claim.Messages():
			log.Write(log.INFO, "Message claimed: ", string(message.Value), message.Timestamp, message.Topic)
			if c.conCurrency <= 1 { //但协程的处理逻辑
				sTime := time.Now()
				if err = c.handle(message.Value); err != nil {
					c.reset(message) //重试逻辑-再次入队列一次
				}
				//处理成功标记完成 否则继续处理
				session.MarkMessage(message, "")
				if !c.autoAck { //手动确认的情况逻辑
					session.Commit()
				}
				log.Write(log.INFO, message.Topic, "任务执行时长:", time.Since(sTime))
			} else {
				c.asyncConsumer(session, message)
			}
		case <-session.Context().Done():
			log.Write(-1, "Kafka ConsumeClaim Done退出逻辑")
			return nil
		}
	}
}
