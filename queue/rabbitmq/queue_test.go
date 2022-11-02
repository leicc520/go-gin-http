package rabbitmq

import (
	"fmt"
	"git.ziniao.com/webscraper/go-gin-http/queue"
	"testing"
	"time"
)

func TestPushv1RabbitMQ(t *testing.T) {
	q := queue.IFQueue(&RabbitMqSt{Url: "amqp://guest:guest@10.100.72.102:5672/", AutoAck: false})
	for i := 0; i < 200; i++ {
		str := fmt.Sprintf("test%08d", i)
		err := q.Publish("test", str)
		fmt.Println(err)
		time.Sleep(time.Second)
	}
}

func TestPushv2RabbitMQ(t *testing.T) {
	q := queue.IFQueue(&RabbitMqSt{Url: "amqp://guest:guest@10.100.72.102:5672/", AutoAck: false})
	for i := 0; i < 200; i++ {
		str := fmt.Sprintf("demo%08d", i)
		err := q.Publish("demo", str)
		fmt.Println(err)
		time.Sleep(time.Second)
	}
}

type ConsumerDemo struct {
	Sleep  time.Duration
	Group  string
	Result bool
}

func (d *ConsumerDemo) Accept(topic string, message []byte) bool {
	fmt.Println(topic, string(message), d.Group)
	time.Sleep(d.Sleep)
	return d.Result //errors.New("Failed")
}

func TestConsumerRabbitMQ(t *testing.T) {
	c := ConsumerDemo{Result: true}
	d := ConsumerDemo{Result: true}
	q := queue.IFQueue(&RabbitMqSt{Url: "amqp://guest:guest@10.100.72.102:5672/", AutoAck: false})
	q.Register("test", &c)
	q.RegisterN("demo", 4, &d)
	q.Start()
}
