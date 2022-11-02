package queue

type PushMessage func(topic string, data interface{}) (err error)

// 消费者要实现的接口逻辑业务
type IFConsumer interface {
	Accept(topic string, message []byte) bool
}

// 消费者结构数据信息
type ConsumerSt struct {
	Consumer    IFConsumer
	Push        PushMessage
	ConCurrency int
	Topic       string
	GoConChan   chan int8
	AutoAck     bool //与应用集成保持一致
}

// 创建一个消费者处理逻辑
func (c *ConsumerSt) Init(conCurrency int, autoAck bool, topic string, h IFConsumer, p PushMessage) {
	c.AutoAck, c.ConCurrency = autoAck, conCurrency
	c.Consumer, c.Push, c.Topic = h, p, topic
	if conCurrency > 1 {
		c.GoConChan = make(chan int8, conCurrency)
	}
}
