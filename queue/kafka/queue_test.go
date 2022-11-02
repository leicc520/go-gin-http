package kafka

import (
	"fmt"
	"git.ziniao.com/webscraper/go-gin-http/queue"
	"testing"
	"time"
)

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

func TestPushv1KafkaMQ(t *testing.T) {
	q := queue.IFQueue(&KafkaMqSt{NodeSrv: "10.100.72.102:9092", Version: "2.8.1", Group: "demo", AutoAck: false})
	for i := 0; i < 200; i++ {
		str := fmt.Sprintf("test%08d", i)
		err := q.Publish("test", str)
		fmt.Println(err)
		time.Sleep(time.Second)
	}
}

func TestPushv2KafkaMQ(t *testing.T) {
	q := queue.IFQueue(&KafkaMqSt{NodeSrv: "10.100.72.102:9092", Version: "2.8.1", Group: "demo", AutoAck: false})
	for i := 0; i < 200; i++ {
		str := fmt.Sprintf("demo%08d", i)
		err := q.Publish("demo", str)
		fmt.Println(err)
		time.Sleep(time.Millisecond * 10)
	}
}

func TestConsumerKafkaMQ(t *testing.T) {
	c := ConsumerDemo{Result: false, Group: "demo", Sleep: time.Second}
	d := ConsumerDemo{Result: true, Group: "demo", Sleep: time.Second}
	q := queue.IFQueue(&KafkaMqSt{NodeSrv: "10.100.72.102:9092", Version: "2.8.1", Group: "demo", AutoAck: false})
	q.Register("test", &c)
	q.RegisterN("demo", 4, &d)
	go func() {
		time.Sleep(time.Second * 3)
		q.Publish("test", "11111111111111111")
	}()
	q.Start()
}

func TestConsumerKafkaMQv2(t *testing.T) {
	c := ConsumerDemo{Result: true, Group: "demov2", Sleep: time.Second}
	d := ConsumerDemo{Result: true, Group: "demov2", Sleep: time.Second}
	q := queue.IFQueue(&KafkaMqSt{NodeSrv: "10.100.72.102:9092", Version: "2.8.1", Group: "demov2", AutoAck: false})
	q.Register("test", &c)
	q.RegisterN("demo", 4, &d)
	q.Start()
}