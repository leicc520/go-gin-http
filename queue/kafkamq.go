package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"git.ziniao.com/webscraper/go-orm/log"
)

/***********************************************************************************************
官方示例文档 https://github.com/Shopify/sarama topic可能需要运维KafkaMQ的人员手动创建
************************************************************************************************/

type KafkaMqSt struct {
	NodeSrv   string   `yaml:"node_srv"`                          //:9092 broker地址,使用都好分割
	AutoAck   bool     `yaml:"auto_ack" default:"true"`           //消费的时候是否自动自动确认
	Group     string   `yaml:"group"`                             //消费的分组编号
	IsSASL    bool     `yaml:"is_sasl" default:"false"`           //是否开启认证
	User      string   `yaml:"user"`                              //链接的账号
	Password  string   `yaml:"password"`                          //链接的密码
	Mechanism string   `yaml:"mechanism" default:"SCRAM-SHA-256"` //认证机制
	Version   string   `yaml:"version" default:"2.8.1"`           //kafka版本号
	Assignor  string   `yaml:"assignor"`                          //partition的获取策略
	Oldest    bool     `yaml:"oldest"`                            //从什么位置开始消费
	Topic     []string `yaml:"topic"`                             //对应topic队列

	l            sync.RWMutex
	config       *sarama.Config
	syncConsumer sarama.ConsumerGroup
	syncProducer sarama.SyncProducer
}

// 释放类库的资源信息
func (r *KafkaMqSt) Close() {
	if r.syncProducer != nil { //生产者
		r.syncProducer.Close()
	}
	if r.syncConsumer != nil { //消费者
		r.syncConsumer.Close()
	}
}

// 初始化Producer数据资料信息
func (r *KafkaMqSt) initSyncProducer() (err error) {
	if r.syncProducer != nil {
		return nil
	}
	nodeSrv := strings.Split(r.NodeSrv, ",")
	config := sarama.NewConfig()
	if len(r.Version) > 0 {
		config.Version, err = sarama.ParseKafkaVersion(r.Version)
		if err != nil {
			log.Write(log.ERROR, "Kafka版本号配置错误", r.Version)
		}
	}
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.Return.Successes = true
	config.Producer.Timeout = 5 * time.Second
	r.syncProducer, err = sarama.NewSyncProducer(nodeSrv, config)
	if err != nil {
		log.Write(log.ERROR, "kafka sync producer start error ", err)
		return err
	}
	return nil
}

// 发送一条消息到队列当中
func (r *KafkaMqSt) Publish(topic string, data interface{}) (err error) {
	//字符串直接转字节数组即可
	var body []byte = nil
	if str, ok := data.(string); ok {
		body = []byte(str)
	} else if body, ok = data.([]byte); !ok {
		body, _ = json.Marshal(data)
	}
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(body),
	}
	r.l.Lock()
	defer func() {
		r.l.Unlock()
		if e := recover(); e != nil {
			err = errors.New(fmt.Sprintf("%+v", e))
		}
	}()
	if err = r.initSyncProducer(); err != nil {
		return err
	}
	_partition, _offset := int32(-1), int64(-1)
	if _partition, _offset, err = r.syncProducer.SendMessage(msg); err != nil {
		log.Write(log.ERROR, "kafka send message error topic{", topic, "}", err)
		return err
	} else {
		log.Write(log.INFO, "kafka send message ok topic{", topic, "}:", _partition, _offset)
	}
	return nil
}

// 队列暂停等逻辑操作
func (r *KafkaMqSt) toggleConsumptionFlow(isPaused *bool) {
	if *isPaused {
		r.syncConsumer.ResumeAll()
		log.Write(log.INFO, "Resuming consumption")
	} else {
		r.syncConsumer.PauseAll()
		log.Write(log.INFO, "Pausing consumption")
	}
	*isPaused = !*isPaused
}

// 初始话生产和消费者逻辑
func (r *KafkaMqSt) consumerConfig() *sarama.Config {
	var err error = nil
	config := sarama.NewConfig()
	if len(r.Version) > 0 {
		config.Version, err = sarama.ParseKafkaVersion(r.Version)
		if err != nil {
			log.Write(log.ERROR, "Kafka版本号配置错误", r.Version)
		}
	}
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	if r.IsSASL {
		config.Net.SASL.Enable = r.IsSASL
		config.Net.SASL.User = r.User
		config.Net.SASL.Password = r.Password
		switch r.Mechanism {
		case "SCRAM-SHA-256":
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case "SCRAM-SHA-512":
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		}
	}
	switch r.Assignor {
	case "sticky":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	case "roundrobin":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	case "range":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	default:
		log.Write(log.ERROR, "Unrecognized consumer group partition assignor: ", r.Assignor)
	}
	if !r.Oldest {
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	}
	if !r.AutoAck { //执行手动确认的处理逻辑
		config.Consumer.Offsets.AutoCommit.Enable = false
	}
	return config
}

// 生成消费队列的业务逻辑处理
func (r *KafkaMqSt) initConsumer(consumer *consumerSt) (err error) {
	keepRunning := true
	ctx, cancel := context.WithCancel(context.Background())
	nodeSrv := strings.Split(r.NodeSrv, ",")
	r.syncConsumer, err = sarama.NewConsumerGroup(nodeSrv, r.Group, r.consumerConfig())
	if err != nil {
		log.Write(log.ERROR, "Error creating consumer group client: %v", err)
		return err
	}
	consumptionIsPaused := false
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err = r.syncConsumer.Consume(ctx, r.Topic, consumer); err != nil {
				log.Write(log.ERROR, "Error from consumer: %v", err)
			}
			log.Write(-1, "Consume 重新进入操作逻辑...", ctx.Err())
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // Await till the consumer has been set up
	log.Write(log.INFO, "Sarama consumer["+r.Group+"] up and running!...")

	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGHUP)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	for keepRunning {
		select {
		case <-ctx.Done():
			log.Write(-1, "terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			log.Write(-1, "terminating: via signal")
			keepRunning = false
		case <-sigusr1:
			r.toggleConsumptionFlow(&consumptionIsPaused)
		}
	}
	cancel()
	wg.Wait()
	if err = r.syncConsumer.Close(); err != nil {
		log.Write(-1, "Error closing client: %v", err)
	}
	if consumer.goConChan != nil { //退出关闭句柄逻辑
		close(consumer.goConChan)
	}
	return nil
}

// 注册到事务处理逻辑
func (r *KafkaMqSt) Consumer(handle QueueCB) error {
	var err error = nil
	consumer := newConsumer(-1, r.AutoAck, handle, r.Publish)
	for { //确实链接不上了退出
		err = r.initConsumer(consumer)
		if err != nil { //关闭重连机制
			log.Write(-1, "kafkamq 重连机制启动...")
			continue
		}
	}
	return err
}

// 并发的处理任务的处理逻辑
func (r *KafkaMqSt) AsyncConsumer(conCurrency int, handle QueueCB) error {
	var err error = nil
	for { //确实链接不上了退出
		consumer := newConsumer(conCurrency, r.AutoAck, handle, r.Publish)
		err = r.initConsumer(consumer)
		if err != nil { //关闭重连机制
			log.Write(-1, "kafkamq 重连机制启动...")
			continue
		}
	}
	return err
}
