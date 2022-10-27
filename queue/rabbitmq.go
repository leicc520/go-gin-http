package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"
	
	"github.com/leicc520/go-orm/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMqSt struct {
	Url string 		`yaml:"url"`
	Durable bool 	`yaml:"durable"`
	AutoAck bool 	`yaml:"auto_ack"`
	AutoDelete bool `yaml:"auto_delete"`
	Queue string 	`yaml:"queue"`
	index  int
	v      int      //版本号管理，每次初始化+1，ack确认需要版本一直,否则丢弃
	isInit bool
	l     sync.RWMutex
	conn *amqp.Connection
	ch   *amqp.Channel
	q     amqp.Queue
}

//设置队列是第几个队列
func (s *RabbitMqSt) SetQueueIndex(idx int) {
	s.index = idx
	log.Write(-1, "开启任务队列{", idx, "}", runtime.NumGoroutine())
}

//往队列发送消息 允许直接传递消息对象指针
func (s *RabbitMqSt) Publish(data interface{}) (err error) {
	var ok = false
	var pMsg *amqp.Publishing = nil
	if pMsg, ok = data.(*amqp.Publishing); !ok {
		//字符串直接转字节数组即可
		var body []byte = nil
		if str, ok := data.(string); ok {
			body = []byte(str)
		} else {
			body, _ = json.Marshal(data)
		}
		pMsg = &amqp.Publishing{ContentType: "text/plain", Body: body}
	}
	s.l.Lock()
	defer func() {
		s.l.Unlock()
		if e := recover(); e != nil {
			err = errors.New(fmt.Sprintf("%+v", e))
		}
	}()
	ctx := context.Background()
	err  = s.ch.PublishWithContext(ctx, "", s.Queue, false, false, *pMsg)
	if err != nil {//失败的情况处理逻辑
		log.Write(log.ERROR, "Failed to publish a message", err)
	}
	return err
}

//克隆一个对象处理逻辑
func (s *RabbitMqSt) Clone(queue string) IFQueue {
	c := &RabbitMqSt{Url: s.Url, Durable: s.Durable,
		AutoAck: s.AutoAck, AutoDelete: s.AutoDelete, Queue:s.Queue+queue}
	return c
}

//初始化队列的链接处理逻辑 初始化的时候要锁定
func (s *RabbitMqSt) Init() error {
	s.l.Lock()
	defer s.l.Unlock()
	if s.isInit {//是否完成初始化,已经完成的需要关闭重来
		s.doClose()
	}
	s.isInit = true
	var err error
	s.conn, err = amqp.Dial(s.Url)
	if err != nil {
		log.Write(log.ERROR, "Failed connect ", err)
		return err
	}
	s.ch, err = s.conn.Channel()
	if err != nil {
		log.Write(log.ERROR, "Failed to open a channel", err)
		return err
	}
	s.q, err = s.ch.QueueDeclare(
		s.Queue, // name
		s.Durable,   // durable
		s.AutoDelete, // delete when unused
		false,   // exclusive
		false,  // no-wait
		nil,     // arguments
	)
	if err != nil {
		log.Write(log.ERROR, "Failed to declare a queue", err)
		return err
	}
	return nil
}

//关闭队列处理逻辑
func (s *RabbitMqSt) doClose()  {
	if s.ch != nil {//关闭渠道
		s.ch.Close()
		s.ch = nil
	}
	if s.conn != nil {//关闭代理
		s.conn.Close()
		s.conn = nil
	}
	s.isInit = false
}

//关闭释放处理逻辑
func (s *RabbitMqSt) Close() {
	s.l.Lock()
	defer s.l.Unlock()
	s.doClose()
}

//确认队列的处理逻辑 释放
func (s *RabbitMqSt) Ack(deliveryTag uint64, version int, err error) (errAck error) {
	s.l.RLock()
	defer s.l.RUnlock()
	//如果是非自动确认的话 需要手动确认 需要版本一直
	if !s.AutoAck && s.v == version {
		if err == nil {
			errAck = s.ch.Ack(deliveryTag, false)
		} else {
			errAck = s.ch.Reject(deliveryTag, true)
		}
	}
	if errAck != nil {
		log.Write(-1, "队列ACK确认异常", errAck)
	}
	log.Write(-1, "队列ACK", s.v, version, err)
	return nil
}

//获取消费队列管道数据信息
func (s *RabbitMqSt) getConsumerChan() (<-chan amqp.Delivery, error) {
	msgChan, err := s.ch.Consume(
		s.q.Name, // queue
		"",     // consumer
		s.AutoAck,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Write(log.ERROR, s.index, s.Queue, "Failed to register a consumer", s.ch.IsClosed(), s.conn.IsClosed(), err)
	}
	return msgChan, err
}

//消费队列数据资料信息
func (s *RabbitMqSt) Consumer(handle QueueCB) error {
	var err error = nil
	for {//确实链接不上了退出
		err = s.consumerStart(handle)
		if err != nil {//关闭重连机制
			if err = s.Init(); err != nil {
				log.Write(-1, s.index, "rabbitmq 重连机制启动失败...", err)
				break
			}
			log.Write(-1, s.index, "rabbitmq 重连机制启动...")
			continue
		}
	}
	return err
}

//开启消费处理逻辑
func (s *RabbitMqSt) consumerStart(handle QueueCB) (err error) {
	var msgChan <-chan amqp.Delivery = nil
	msgChan, err = s.getConsumerChan()
	if err != nil {
		return err
	}
	defer func() {
		if e := recover(); e != nil {
			err = errors.New(fmt.Sprintf("%+v", e))
		}
	}()
	for {
		dMsg, isClose := <-msgChan
		if !isClose {//异常的处理逻辑，不一定是真的关闭了链接
			log.Write(-1, s.index, s.Queue, "queue closed", s.ch.IsClosed(), s.conn.IsClosed(), dMsg)
			if s.ch.IsClosed() {
				err = errors.New(s.Queue+" queue closed")
				break
			}
			continue
		}
		log.Write(log.INFO, s.index, s.Queue, " Received a message: ", string(dMsg.Body), err)
		err = handle(dMsg.Body) //业务只需关注数据即可
		s.Ack(dMsg.DeliveryTag, s.v, err) //处理完结确认
	}
	return err
}

//消费队列数据资料信息
func (s *RabbitMqSt) AsyncConsumer(conCurrency int, handle QueueCB) error {
	var err error = nil
	//设置缓冲区 以便于控制开启协程的数量，限制并发
	goConChan := make(chan int8, conCurrency)
	defer func() {//退出关闭逻辑
		close(goConChan)
	}()
	for {//确实链接不上了退出
		err = s.asyncConsumerStart(goConChan, s.v, handle)
		if err != nil {//关闭重连机制
			if err = s.Init(); err != nil {
				log.Write(-1, s.index, conCurrency, "rabbitmq 重连机制启动失败...", err)
				break
			}
			s.v++ //重启之后版本+1
			log.Write(-1, s.index, conCurrency, "rabbitmq 重连机制启动...", runtime.NumGoroutine())
			continue
		}
	}
	return err
}

//启动异步执行处理逻辑
func (s *RabbitMqSt) asyncConsumerStart(goConChan chan int8, version int, handle QueueCB) (err error) {
	var msgChan <-chan amqp.Delivery = nil
	msgChan, err = s.getConsumerChan()
	if err != nil {
		return err
	}
	defer func() {
		if e := recover(); e != nil {
			log.Write(-1, s.index, s.Queue, "asyncConsumer 结束异常", e)
			err = errors.New(fmt.Sprintf("%+v", e))
		}
	}()
	for {
		dMsg, isClose := <-msgChan
		if !isClose {//有的时候会异常关闭的情况逻辑
			log.Write(-1, s.index, s.Queue, "queue closed", s.ch.IsClosed(), s.conn.IsClosed(), dMsg)
			if s.ch.IsClosed() {//管道还在的话继续跑
				err = errors.New(s.Queue+" queue closed")
				break
			}
			continue
		}
		goConChan <- 1
		go func(dlMsg amqp.Delivery, v int, goSConChan chan int8) {
			defer func() { //结束释放并发位置
				<-goSConChan
				if e := recover(); e != nil {
					log.Write(-1, s.index, s.Queue, "queue handle panic", e)
				}
			}()
			log.Write(log.INFO, s.index, s.Queue, " Received a message: ", string(dMsg.Body), err)
			sTime := time.Now()
			err = handle(dlMsg.Body) //业务只需关注数据即可
			log.Write(log.INFO, s.index, s.Queue, "任务执行时长:", time.Since(sTime))
			s.Ack(dlMsg.DeliveryTag, v, err) //处理完结确认
		}(dMsg, version, goConChan)
	}
	return err
}
