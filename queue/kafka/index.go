package kafka

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"git.ziniao.com/webscraper/go-gin-http/queue"
	"git.ziniao.com/webscraper/go-orm/log"
	"github.com/Shopify/sarama"
)

/***********************************************************************************************
官方示例文档 https://github.com/Shopify/sarama topic可能需要运维KafkaMQ的人员手动创建
************************************************************************************************/

type KafkaMqSt struct {
	NodeSrv       string               `yaml:"node_srv"`                          //:9092 broker地址,使用都好分割
	AutoAck       bool                 `yaml:"auto_ack" default:"true"`           //消费的时候是否自动自动确认
	Group         string               `yaml:"group"`                             //消费的分组编号
	IsSASL        bool                 `yaml:"is_sasl" default:"false"`           //是否开启认证
	User          string               `yaml:"user"`                              //链接的账号
	Password      string               `yaml:"password"`                          //链接的密码
	Mechanism     string               `yaml:"mechanism" default:"SCRAM-SHA-256"` //认证机制
	Version       string               `yaml:"version" default:"2.8.1"`           //kafka版本号
	Assignor      string               `yaml:"assignor"`                          //partition的获取策略
	Oldest        bool                 `yaml:"oldest"`                            //从什么位置开始消费
	queue.QueueSt                      //集成基础结构体
	syncProducer  sarama.SyncProducer  `yaml:"-"` //主要用作发信息
	syncConsumer  sarama.ConsumerGroup `yaml:"-"` //消费者群组
}

// 初始化处理逻辑,需要执行一次即可
func (s *KafkaMqSt) initConsumer() (err error) {
	s.Once.Do(func() {
		if s.Topics == nil {
			s.Topics = make(queue.TopicConsumeSt)
		}
	})
	config := s._config()
	s.ConsumerCtx, s.ConsumerCancel = context.WithCancel(context.Background())
	nodeSrv := strings.Split(s.NodeSrv, ",")
	s.syncConsumer, err = sarama.NewConsumerGroup(nodeSrv, s.Group, config)
	if err != nil { //创建消费组失败的情况
		log.Write(log.ERROR, "Error creating consumer group client: %v", err)
	}
	return err
}

// 关闭链接的处理逻辑
func (s *KafkaMqSt) Close() {
	s.L.Lock()
	defer s.L.Unlock()
	s.closeProducer(false)
	//结束任务
	if s.ConsumerCancel != nil {
		s.ConsumerCancel()
		s.ConsumerCancel = nil
	}
	if s.syncConsumer != nil {
		s.syncConsumer.Close()
		s.syncConsumer = nil
	}
	log.Write(-1, "kafka队列执行退出处理逻辑...")
}

// 释放类库的资源信息
func (s *KafkaMqSt) closeProducer(isSleep bool) {
	if s.syncProducer != nil { //生产者
		s.syncProducer.Close()
		s.syncProducer = nil
		if isSleep { //休眠一下，然后重试
			time.Sleep(time.Second)
		}
	}
}

// 初始化Producer数据资料信息
func (s *KafkaMqSt) initSyncProducer() (err error) {
	if s.syncProducer != nil {
		return nil
	}
	nodeSrv := strings.Split(s.NodeSrv, ",")
	config := sarama.NewConfig()
	if len(s.Version) > 0 {
		config.Version, err = sarama.ParseKafkaVersion(s.Version)
		if err != nil {
			log.Write(log.ERROR, "Kafka版本号配置错误", s.Version)
		}
	}
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.Return.Successes = true
	config.Producer.Timeout = 5 * time.Second
	s.syncProducer, err = sarama.NewSyncProducer(nodeSrv, config)
	if err != nil {
		log.Write(log.ERROR, "kafka sync producer start error ", err)
		return err
	}
	return nil
}

// 发送一条消息到队列当中
func (s *KafkaMqSt) Publish(topic string, data interface{}) (err error) {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(s.Format(data)),
	}
	s.L.Lock()
	defer s.L.Unlock()
	_partition, _offset := int32(-1), int64(-1)
	for i := 0; i < queue.RetryLimit; i++ { //发布消息失败重试3次的处理逻辑
		if err = s.initSyncProducer(); err != nil { //连接失败等待一秒重连
			s.closeProducer(true)
			continue
		}
		_partition, _offset, err = s.syncProducer.SendMessage(msg)
		if err != nil {
			if i == 2 { //执行一次重连试试
				s.closeProducer(true)
			}
			continue
		}
		break
	}
	log.Write(log.INFO, "kafka发送消息{", topic, "}:", _partition, _offset, err)
	return
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
func (s *KafkaMqSt) _config() *sarama.Config {
	var err error = nil
	config := sarama.NewConfig()
	if len(s.Version) > 0 {
		config.Version, err = sarama.ParseKafkaVersion(s.Version)
		if err != nil {
			log.Write(log.ERROR, "Kafka版本号配置错误", s.Version)
		}
	}
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	if s.IsSASL {
		config.Net.SASL.Enable = s.IsSASL
		config.Net.SASL.User = s.User
		config.Net.SASL.Password = s.Password
		switch s.Mechanism {
		case "SCRAM-SHA-256":
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case "SCRAM-SHA-512":
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		}
	}
	switch s.Assignor {
	case "sticky":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	case "roundrobin":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	case "range":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	default:
		log.Write(log.INFO, "Unrecognized consumer group partition assignor: ", s.Assignor)
	}
	if s.Oldest {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}
	if !s.AutoAck { //执行手动确认的处理逻辑
		config.Consumer.Offsets.AutoCommit.Enable = false
	}
	return config
}

// 开始一个服务处理逻辑
func (s *KafkaMqSt) Start(h func()) (err error) {
	h()              //回调注册处理逻辑
	s.initConsumer() //初始化完成
	keepRunning := true
	consumptionIsPaused := false
	regAccept := make(map[string]*kafkaConsumerSt)
	for topic, cWrapper := range s.Topics { //创建消费者对象逻辑
		regAccept[topic] = NewKafkaConsumer(cWrapper.ConCurrency,
			s.AutoAck, topic, cWrapper.Handle, s.Publish)
	}
	cc := s.consumerStart(regAccept) //启动消费服务监听
	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGHUP)
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	for keepRunning {
		select {
		case <-s.ConsumerCtx.Done():
			log.Write(-1, "terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			log.Write(-1, "terminating: via signal")
			keepRunning = false
		case <-sigusr1:
			s.toggleConsumptionFlow(&consumptionIsPaused)
		}
	}
	s.ConsumerCancel()
	s.ConsumerWg.Wait()
	s.Close()  //执行退出了
	cc.Close() //关闭句柄数据信息
	return nil
}

// 启动一个消费者处理逻辑业务
func (s *KafkaMqSt) consumerStart(regAccept map[string]*kafkaConsumerSt) *consumeClaimSt {
	s.ConsumerWg.Add(1)
	cc := NewKafkaConsumeClaim(regAccept)
	go func() {
		defer s.ConsumerWg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := s.syncConsumer.Consume(s.ConsumerCtx, s.GetTopics(), cc); err != nil {
				log.Write(log.ERROR, "Error from consumer: %v", err)
			}
			log.Write(-1, "Consume 重新进入操作逻辑...", s.ConsumerCtx.Err())
			// check if context was cancelled, signaling that the consumer should stop
			if s.ConsumerCtx.Err() != nil {
				return
			}
			cc.Ready = make(chan bool)
		}
	}()
	<-cc.Ready // Await till the consumer has been set up
	log.Write(log.INFO, "Sarama consumer["+strings.Join(s.GetTopics(), ",")+"] up and running!...")
	return cc
}
