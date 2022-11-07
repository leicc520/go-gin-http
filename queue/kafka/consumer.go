package kafka

import (
	"time"

	"git.ziniao.com/webscraper/go-gin-http/queue"
	"git.ziniao.com/webscraper/go-orm/log"
	"github.com/Shopify/sarama"
)

// 消费者结构数据信息
type kafkaConsumerSt struct {
	queue.ConsumerSt
}

// 创建一个消费者处理逻辑
func NewKafkaConsumer(conCurrency int, autoAck bool, topic string, h queue.IFConsumer, p queue.PushMessage) *kafkaConsumerSt {
	c := &kafkaConsumerSt{}
	c.Init(conCurrency, autoAck, topic, h, p)
	return c
}

// 重试机制的出来逻辑
func (c *kafkaConsumerSt) reset(message *sarama.ConsumerMessage) bool {
	if c.Push != nil { //更新数据信息
		if err := c.Push(message.Topic, message.Value); err != nil {
			log.Write(log.DEBUG, "Kafka重试重新入队列异常", err)
			return false
		}
	}
	return true
}

// 异步消费的处理逻辑
func (c *kafkaConsumerSt) asyncConsumer(message *sarama.ConsumerMessage, session sarama.ConsumerGroupSession) {
	c.GoConChan <- 1
	go func(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
		defer func() {
			<-c.GoConChan
			if e := recover(); e != nil {
				log.Write(-1, message.Topic, "Kafka queue handle panic", e)
			}
		}()
		c.syncConsumer(message, session) //同步消费逻辑
	}(session, message)
}

// 同步的执行任务处理业务逻辑
func (c *kafkaConsumerSt) syncConsumer(message *sarama.ConsumerMessage, session sarama.ConsumerGroupSession) {
	sTime := time.Now()
	if isOk := c.Consumer.Accept(message.Topic, message.Value); !isOk {
		if !c.reset(message) { //重试逻辑-再次入队列一次
			return
		}
	}
	log.Write(log.INFO, message.Topic, "任务执行时长:", time.Since(sTime))
	//处理成功标记完成 否则继续处理
	session.MarkMessage(message, "")
	if !c.AutoAck { //手动确认的情况逻辑
		session.Commit()
	}
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *kafkaConsumerSt) consumerMessage(message *sarama.ConsumerMessage, session sarama.ConsumerGroupSession) {
	defer func() {
		if err := recover(); err != nil {
			log.Write(log.DEBUG, "消息业务逻辑处理失败", err)
		}
	}()
	if c.ConCurrency <= 1 { //但协程的处理逻辑
		c.syncConsumer(message, session)
	} else {
		c.asyncConsumer(message, session)
	}
}
