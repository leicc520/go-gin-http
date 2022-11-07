package kafka

import (
	"git.ziniao.com/webscraper/go-orm/log"
	"github.com/Shopify/sarama"
)

// 消费者结构数据信息
type consumeClaimSt struct {
	regConsumer map[string]*kafkaConsumerSt
	Ready       chan bool
}

// 创建一个消费者处理逻辑
func NewKafkaConsumeClaim(reg map[string]*kafkaConsumerSt) *consumeClaimSt {
	c := &consumeClaimSt{Ready: make(chan bool), regConsumer: reg}
	return c
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *consumeClaimSt) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(c.Ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *consumeClaimSt) Cleanup(sarama.ConsumerGroupSession) error {
	log.Write(log.DEBUG, "Claim清理数据", c.regConsumer)
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *consumeClaimSt) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
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

// 关闭句柄数据资料信息
func (c *consumeClaimSt) Close() {
	if c.regConsumer != nil { //并发处理的话关闭句柄
		for _, regAccept := range c.regConsumer {
			if regAccept.ConCurrency > 1 {
				close(regAccept.GoConChan)
			}
		}
	}
}
