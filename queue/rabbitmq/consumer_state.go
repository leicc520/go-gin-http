package rabbitmq

import (
	"git.ziniao.com/webscraper/go-orm/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

// 获取rabbitmq状态
type rabbitMqStateSt struct {
	Version int //版本号管理，每次初始化+1，ack确认需要版本一直,否则丢弃
	Channel *amqp.Channel
	Queue   amqp.Queue
}

// 关闭管道处理逻辑
func (s *rabbitMqStateSt) Close() {
	if s.Channel != nil { //关闭渠道
		s.Channel.Close()
		s.Channel = nil
	}
}

// 初始化创建一个消息管道
func (s *rabbitMqStateSt) Init(topic string, durable, autoDelete bool, conn *amqp.Connection) (err error) {
	s.Channel, err = conn.Channel()
	if err != nil {
		log.Write(log.ERROR, "Failed to open a channel", err)
		return err
	}
	s.Queue, err = s.Channel.QueueDeclare(
		topic,      // name
		durable,    // durable
		autoDelete, // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		log.Write(log.ERROR, "Failed to declare a queue", err)
		return err
	}
	return
}

// 获取消费队列管道数据信息
func (s *rabbitMqStateSt) consumer(autoAck bool) (<-chan amqp.Delivery, error) {
	msgChan, err := s.Channel.Consume(
		s.Queue.Name, // queue
		"",           // consumer
		autoAck,      // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		log.Write(log.ERROR, s.Queue.Name, "Failed to register a consumer", s.Channel.IsClosed(), err)
	}
	return msgChan, err
}
