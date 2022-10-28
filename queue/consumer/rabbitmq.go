package consumer

// 消费者结构数据信息
type RabbitMqConsumerSt struct {
	consumerSt
}

// 创建一个消费者处理逻辑
func NewRabbitMqConsumer(conCurrency int, autoAck bool, h IFConsumer, p pMessage) *RabbitMqConsumerSt {
	c := &RabbitMqConsumerSt{}
	c.init(conCurrency, autoAck, h, p)
	return c
}
