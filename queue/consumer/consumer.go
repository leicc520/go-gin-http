package consumer

type pMessage func(topic string, data interface{}) (err error)

// 消费者要实现的接口逻辑业务
type IFConsumer interface {
	Accept(topic string, message []byte) bool
}

// 消费者结构数据信息
type consumerSt struct {
	consumer    IFConsumer
	push        pMessage
	conCurrency int
	topic       string
	goConChan   chan int8
	autoAck     bool //与应用集成保持一致
}

// 创建一个消费者处理逻辑
func (c *consumerSt) init(conCurrency int, autoAck bool, topic string, h IFConsumer, p pMessage) {
	c.autoAck, c.conCurrency = autoAck, conCurrency
	c.consumer, c.push, c.topic = h, p, topic
	if conCurrency > 1 {
		c.goConChan = make(chan int8, conCurrency)
	}
}
