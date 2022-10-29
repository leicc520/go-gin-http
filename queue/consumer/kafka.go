package consumer

import (
	"git.ziniao.com/webscraper/go-orm/log"
	"github.com/Shopify/sarama"
)

// 消费者结构数据信息
type KafkaConsumeClaimSt struct {
	regConsumer map[string]*KafkaConsumerSt
	Ready       chan bool
}

// 创建一个消费者处理逻辑
func NewKafkaConsumeClaim(reg map[string]*KafkaConsumerSt) *KafkaConsumeClaimSt {
	c := &KafkaConsumeClaimSt{Ready: make(chan bool), regConsumer: reg}
	return c
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *KafkaConsumeClaimSt) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(c.Ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *KafkaConsumeClaimSt) Cleanup(sarama.ConsumerGroupSession) error {
	if c.regConsumer != nil { //并发处理的话关闭句柄
		for _, regAccept := range c.regConsumer {
			if regAccept.conCurrency > 1 {
				close(regAccept.goConChan)
			}
		}
	}
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *KafkaConsumeClaimSt) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			log.Write(log.INFO, "Message claimed: ", string(message.Value), message.Timestamp, message.Topic)
			if regConsumer, ok := c.regConsumer[message.Topic]; ok {
				regConsumer.consumerMessage(message, session)
			} else { //未注册的情况逻辑
				log.Write(-1, "丢弃未注册的Topic【"+message.Topic+"】任务逻辑...")
			}
		case <-session.Context().Done():
			log.Write(-1, "Kafka ConsumeClaim Done退出逻辑")
			return nil
		}
	}
	return nil
}
