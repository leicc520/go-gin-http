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

func (d *ConsumerDemo) Accept(topic string, message []byte) bool {
	fmt.Println(topic, string(message))
	return false //errors.New("Failed")
}

func TestConsumerRabbitMQ(t *testing.T) {
	c := ConsumerDemo{}
	d := ConsumerDemo{}
	q := IFQueue(&RabbitMqSt{Url: "amqp://guest:guest@192.168.138.128:5673/", AutoAck: false})
	q.Register("test", &c)
	q.RegisterN("demo", 4, &d)
	q.Start()
}

func TestPushv1KafkaMQ(t *testing.T) {
	q := IFQueue(&KafkaMqSt{NodeSrv: "192.168.138.128:9092", Version: "2.7.0", Group: "demo", AutoAck: false})
	for i := 0; i < 200; i++ {
		str := fmt.Sprintf("test%08d", i)
		err := q.Publish("test", str)
		fmt.Println(err)
		time.Sleep(time.Second)
	}
}

func TestPushv2KafkaMQ(t *testing.T) {
	q := IFQueue(&KafkaMqSt{NodeSrv: "192.168.138.128:9092", Version: "2.7.0", Group: "demo", AutoAck: false})
	for i := 0; i < 200; i++ {
		str := fmt.Sprintf("demo%08d", i)
		err := q.Publish("demo", str)
		fmt.Println(err)
		time.Sleep(time.Second)
	}
}

func TestConsumerKafkaMQ(t *testing.T) {
	c := ConsumerDemo{}
	d := ConsumerDemo{}
	q := IFQueue(&KafkaMqSt{NodeSrv: "192.168.138.128:9092", Version: "2.7.0", Group: "demo", AutoAck: false})
	q.Register("test", &c)
	q.RegisterN("demo", 4, &d)
	go func() {
		time.Sleep(time.Second * 3)
		q.Publish("test", "11111111111111111")
	}()
	q.Start()
}
