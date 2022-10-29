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
}

// 创建一个消费者处理逻辑
func NewKafkaConsumer(conCurrency int, autoAck bool, topic string, h IFConsumer, p pMessage) *KafkaConsumerSt {
	c := &KafkaConsumerSt{}
	c.init(conCurrency, autoAck, topic, h, p)
	return c
}

// 重试机制的出来逻辑
func (c *KafkaConsumerSt) reset(message *sarama.ConsumerMessage) {
	var err = errors.New("重试调用未执行")
	if c.push != nil { //更新数据信息
		err = c.push(message.Topic, message.Value)
	}
	log.Write(log.INFO, "Kafka重试机制", err)
}

// 异步消费的处理逻辑
func (c *KafkaConsumerSt) asyncConsumer(message *sarama.ConsumerMessage, session sarama.ConsumerGroupSession) {
	c.goConChan <- 1
	go func(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
		defer func() {
			<-c.goConChan
			if e := recover(); e != nil {
				log.Write(-1, message.Topic, "Kafka queue handle panic", e)
			}
		}()

		sTime := time.Now() //统计任务执行时长
		if isOk := c.consumer.Accept(message.Topic, message.Value); !isOk {
			c.reset(message) //重试逻辑-再次入队列一次
		}
		log.Write(log.INFO, message.Topic, "Kafka任务执行时长:", time.Since(sTime))
		session.MarkMessage(message, "")
		if !c.autoAck { //手动确认的情况逻辑
			session.Commit()
		}
	}(session, message)
}

// 同步的执行任务处理业务逻辑
func (c *KafkaConsumerSt) syncConsumer(message *sarama.ConsumerMessage, session sarama.ConsumerGroupSession) {
	sTime := time.Now()
	if isOk := c.consumer.Accept(message.Topic, message.Value); !isOk {
		c.reset(message) //重试逻辑-再次入队列一次
	}
	log.Write(log.INFO, message.Topic, "任务执行时长:", time.Since(sTime))
	//处理成功标记完成 否则继续处理
	session.MarkMessage(message, "")
	if !c.autoAck { //手动确认的情况逻辑
		session.Commit()
	}
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *KafkaConsumerSt) consumerMessage(message *sarama.ConsumerMessage, session sarama.ConsumerGroupSession) {
	if c.conCurrency <= 1 { //但协程的处理逻辑
		c.syncConsumer(message, session)
	} else {
		c.asyncConsumer(message, session)
	}
}
