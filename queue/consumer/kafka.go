package consumer

import (
	"errors"
	"time"

	"git.ziniao.com/webscraper/go-orm/log"
	"github.com/Shopify/sarama"
)

// 消费者结构数据信息
type KafkaConsumerSt struct {
	consumerSt
	Ready chan bool
}

// 创建一个消费者处理逻辑
func NewKafkaConsumer(conCurrency int, autoAck bool, h IFConsumer, p pMessage) *KafkaConsumerSt {
	c := &KafkaConsumerSt{Ready: make(chan bool)}
	c.init(conCurrency, autoAck, h, p)
	return c
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *KafkaConsumerSt) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(c.Ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *KafkaConsumerSt) Cleanup(sarama.ConsumerGroupSession) error {
	if c.conCurrency > 1 { //并发处理的话关闭句柄
		close(c.goConChan)
	}
	return nil
}

// 重试机制的出来逻辑
func (c *consumerSt) reset(message *sarama.ConsumerMessage) {
	var err = errors.New("重试调用未执行")
	if c.push != nil { //更新数据信息
		err = c.push(message.Topic, message.Value)
	}
	log.Write(log.INFO, "Kafka重试机制", err)
}

// 异步消费的处理逻辑
func (c *consumerSt) asyncConsumer(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	var err error = nil
	c.goConChan <- 1
	go func(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
		defer func() {
			<-c.goConChan
			if e := recover(); e != nil {
				log.Write(-1, message.Topic, "Kafka queue handle panic", e)
			}
		}()

		sTime := time.Now() //统计任务执行时长
		if err = c.consumer.Accept(message.Topic, message.Value); err != nil {
			c.reset(message) //重试逻辑-再次入队列一次
		}
		session.MarkMessage(message, "")
		if !c.autoAck { //手动确认的情况逻辑
			session.Commit()
		}
		log.Write(log.INFO, message.Topic, "Kafka任务执行时长:", time.Since(sTime))

	}(session, message)
}

// 同步的执行任务处理业务逻辑
func (c *KafkaConsumerSt) syncConsumer(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	sTime := time.Now()
	if err := c.consumer.Accept(message.Topic, message.Value); err != nil {
		c.reset(message) //重试逻辑-再次入队列一次
	}
	//处理成功标记完成 否则继续处理
	session.MarkMessage(message, "")
	if !c.autoAck { //手动确认的情况逻辑
		session.Commit()
	}
	log.Write(log.INFO, message.Topic, "任务执行时长:", time.Since(sTime))
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *KafkaConsumerSt) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			log.Write(log.INFO, "Message claimed: ", string(message.Value), message.Timestamp, message.Topic)
			if c.conCurrency <= 1 { //但协程的处理逻辑
				c.asyncConsumer(session, message)
			} else {
				c.asyncConsumer(session, message)
			}
		case <-session.Context().Done():
			log.Write(-1, "Kafka ConsumeClaim Done退出逻辑")
			return nil
		}
	}
}
