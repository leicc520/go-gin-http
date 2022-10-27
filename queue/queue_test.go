package queue

import (
	"fmt"
	"github.com/leicc520/go-orm"
	"github.com/leicc520/go-orm/log"
	"strconv"
	"testing"
	"time"
)


func TestRetry(t *testing.T) {
	q := IFQueue(&RabbitMqSt{Url: "amqp://guest:guest@10.100.72.102:5672/", Queue: "demo"})
	q.Init()
	go func() {
		for {
			time.Sleep(time.Second*1)
			q.Close()
		}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			q.Publish("111111=="+strconv.FormatInt(int64(i), 10))
			time.Sleep(time.Millisecond*500)
		}
	}()
	err := q.Consumer(func(bytes []byte) error {
		fmt.Println(string(bytes))
		return nil
	})
	fmt.Println(err)
}

func TestQueue(t *testing.T) {
	q := IFQueue(&RabbitMqSt{Url: "amqp://guest:guest@10.100.72.102:5672/", Queue: "demo"})

	err := q.Init()
	if err != nil {
		return
	}
	defer q.Close()
	cb := func([]byte) error { return nil}
	go q.AsyncConsumer(3, cb)

	sp := func(i int, max int) {
		for i < max {
			q.Publish(orm.SqlMap{"data":i})
			log.Write(log.INFO, i)
			i++
			time.Sleep(time.Millisecond*50)
		}
	}

	go sp(1, 1000000)
	go sp(30, 50)
	go sp(1, 20)
	go sp(30, 50)
	go sp(1, 20)
	go sp(30, 50)
	go sp(1, 20)
	go sp(30, 50)
	go sp(1, 20)
	go sp(30, 50)
	go sp(1, 20)
	go sp(30, 50)

	sp(60, 100)

	c := make(chan int)
	<-c
}
