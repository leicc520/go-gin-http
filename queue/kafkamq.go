package queue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	
	"git.ziniao.com/webscraper/go-gin-http/queue/consumer"
	"git.ziniao.com/webscraper/go-orm/log"
	"github.com/Shopify/sarama"
)

/***********************************************************************************************
官方示例文档 https://github.com/Shopify/sarama topic可能需要运维KafkaMQ的人员手动创建
************************************************************************************************/

type KafkaMqSt struct {
	NodeSrv             string               `yaml:"node_srv"`                          //:9092 broker地址,使用都好分割
	AutoAck             bool                 `yaml:"auto_ack" default:"true"`           //消费的时候是否自动自动确认
	Group               string               `yaml:"group"`                             //消费的分组编号
	IsSASL              bool                 `yaml:"is_sasl" default:"false"`           //是否开启认证
	User                string               `yaml:"user"`                              //链接的账号
	Password            string               `yaml:"password"`                          //链接的密码
	Mechanism           string               `yaml:"mechanism" default:"SCRAM-SHA-256"` //认证机制
	Version             string               `yaml:"version" default:"2.8.1"`           //kafka版本号
	Assignor            string               `yaml:"assignor"`                          //partition的获取策略
	Oldest              bool                 `yaml:"oldest"`                            //从什么位置开始消费
	QueueSt                                                                             //集成基础结构体
	syncProducer        sarama.SyncProducer  `yaml:"-"`                                 //主要用作发信息
	syncConsumer        sarama.ConsumerGroup `yaml:"-"`                                 //消费者群组
}

// 初始化处理逻辑,需要执行一次即可
func (r *KafkaMqSt) Init() {
	r.once.Do(func() {
		r.topics = make(TopicConsumeSt)
	})
}

// 关闭链接的处理逻辑
func (r *KafkaMqSt) Close() {
	r.l.Lock()
	defer r.l.Unlock()
	r.closeProducer(false)
	if r.syncConsumer != nil {
		r.syncConsumer.Close()
		r.syncConsumer = nil
	}
	log.Write(-1, "kafka队列执行退出处理逻辑...")
}

// 释放类库的资源信息
func (r *KafkaMqSt) closeProducer(isSleep bool) {
	if r.syncProducer != nil { //生产者
		r.syncProducer.Close()
		r.syncProducer = nil
		if isSleep { //休眠一下，然后重试
			time.Sleep(time.Second)
		}
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
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(r.format(data)),
	}
	r.l.Lock()
	defer func() {
		r.l.Unlock()
		if e := recover(); e != nil {
			err = errors.New(fmt.Sprintf("%+v", e))
		}
	}()
	_partition, _offset := int32(-1), int64(-1)
	for i := 0; i < retryLimit; i++ { //发布消息失败重试3次的处理逻辑
		if err = r.initSyncProducer(); err != nil { //连接失败等待一秒重连
			r.closeProducer(true)
			continue
		}
		_partition, _offset, err = r.syncProducer.SendMessage(msg)
		if err != nil {
			if i == 2 { //执行一次重连试试
				r.closeProducer(true)
			}
			continue
		}
		break
	}
	if err != nil {
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
		log.Write(log.INFO, "Kafka Resuming consumption")
	} else {
		r.syncConsumer.PauseAll()
		log.Write(log.INFO, "Kafka Pausing consumption")
	}
	*isPaused = !*isPaused
}

// 初始话生产和消费者逻辑
func (r *KafkaMqSt) _config() *sarama.Config {
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

// 开始一个服务处理逻辑
func (r *KafkaMqSt) Start() (err error) {
	r.Init()//初始化完成
	config := r._config()
	keepRunning := true
	r.consumerCtx, r.consumerCancel = context.WithCancel(context.Background())
	nodeSrv := strings.Split(r.NodeSrv, ",")
	r.syncConsumer, err = sarama.NewConsumerGroup(nodeSrv, r.Group, config)
	if err != nil { //创建消费组失败的情况
		log.Write(log.ERROR, "Error creating consumer group client: %v", err)
		return err
	}
	consumptionIsPaused := false
	//遍历注册consumer到消费组当中
	for topic, consumerWrapper := range r.topics {
		r.consumerStart(topic, &consumerWrapper)
	}
	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGHUP)
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	for keepRunning {
		select {
		case <-r.consumerCtx.Done():
			log.Write(-1, "terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			log.Write(-1, "terminating: via signal")
			keepRunning = false
		case <-sigusr1:
			r.toggleConsumptionFlow(&consumptionIsPaused)
		}
	}
	r.consumerCancel()
	r.consumerWg.Wait()
	r.Close() //执行退出了
	return nil
}

// 启动一个消费者处理逻辑业务
func (r *KafkaMqSt) consumerStart(topic string, cWrapper *consumerWrapperSt) {
	r.consumerWg.Add(1)
	cc := consumer.NewKafkaConsumer(cWrapper.conCurrency, r.AutoAck, topic, cWrapper.handle, r.Publish)
	go func() {
		defer r.consumerWg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := r.syncConsumer.Consume(r.consumerCtx, []string{topic}, cc); err != nil {
				log.Write(log.ERROR, "Error from consumer: %v", err)
			}
			log.Write(-1, "Consume 重新进入操作逻辑...", r.consumerCtx.Err())
			// check if context was cancelled, signaling that the consumer should stop
			if r.consumerCtx.Err() != nil {
				return
			}
			cc.Ready = make(chan bool)
		}
	}()
	<-cc.Ready // Await till the consumer has been set up
	log.Write(log.INFO, "Sarama consumer["+topic+"] up and running!...")
}
