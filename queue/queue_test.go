package queue

import (
	"fmt"
	"testing"
	"time"
)

func TestPushv1RabbitMQ(t *testing.T) {
	q := IFQueue(&RabbitMqSt{Url: "amqp://guest:guest@192.168.138.128:5673/", AutoAck: false})
	for i := 0; i < 200; i++ {
		str := fmt.Sprintf("test%08d", i)
		err := q.Publish("test", str)
		fmt.Println(err)
		time.Sleep(time.Second)
	}
}

func TestPushv2RabbitMQ(t *testing.T) {
	q := IFQueue(&RabbitMqSt{Url: "amqp://guest:guest@192.168.138.128:5673/", AutoAck: false})
	for i := 0; i < 200; i++ {
		str := fmt.Sprintf("demo%08d", i)
		err := q.Publish("demo", str)
		fmt.Println(err)
		time.Sleep(time.Second)
	}
}

type ConsumerDemo struct {
}

func (d *ConsumerDemo) Accept(topic string, message []byte) error {
	fmt.Println(topic, message)
	return nil
}

func TestConsumerRabbitMQ(t *testing.T) {
	c := ConsumerDemo{}
	d := ConsumerDemo{}
	q := IFQueue(&RabbitMqSt{Url: "amqp://guest:guest@192.168.138.128:5673/", AutoAck: false})
	q.Register("test", &c)
	q.RegisterN("demo", 4, &d)
	q.Start()
}
