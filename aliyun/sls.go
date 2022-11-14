package aliyun

import (
	"time"

	"git.ziniao.com/webscraper/go-orm/log"
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/aliyun/aliyun-log-go-sdk/producer"
	"google.golang.org/protobuf/proto"
)

var (
	SLSConfig   *AliYunSLSSt       = nil
	SLSProducer *producer.Producer = nil
)

type AliYunSLSSt struct {
	EndPint         string `json:"end_pint"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	SLSStoreBucket  string `json:"sls_store_bucket"`
	SLSTopic        string `json:"sls_topic"`
	SLSProject      string `json:"sls_project"`
}

// 初始化
func (s *AliYunSLSSt) InitProducer() {
	config := producer.GetDefaultProducerConfig()
	config.Endpoint = s.EndPint
	config.AccessKeyID = s.AccessKeyID
	config.AccessKeySecret = s.AccessKeySecret
	config.AllowLogLevel = "error"
	SLSConfig = s //配置注册到引用里面
	SLSProducer = producer.InitProducer(config)
	SLSProducer.Start() //启动日志收集服务
}

type SLSCallback struct{}

func (cb *SLSCallback) Success(result *producer.Result) {
	log.Write(log.INFO, "send sls log fail", result.GetErrorMessage())
}

func (cb *SLSCallback) Fail(result *producer.Result) {
	log.Write(log.ERROR, "send sls log fail", result.GetErrorMessage())
}

// 发送记录sls日志请求
func SendSLSLog(contents []*sls.LogContent) {
	if SLSProducer == nil || SLSConfig == nil {
		log.Write(log.DEBUG, "send sls log not init")
		return
	}
	err := SLSProducer.SendLogWithCallBack(SLSConfig.SLSProject, SLSConfig.SLSStoreBucket, SLSConfig.SLSTopic, "", &sls.Log{
		Time:     proto.Uint32(uint32(time.Now().Unix())),
		Contents: contents,
	}, &SLSCallback{})
	if err != nil {
		log.Write(log.ERROR, "send sls log", err)
	}
}

// 关闭SLS日志链接请求
func CloseSLSLog() {
	if SLSProducer != nil {
		SLSProducer.SafeClose()
	}
}

// 包装日志数据信息结构处理逻辑
func SLSWrapped(node string, content string, category string) []*sls.LogContent {
	return []*sls.LogContent{
		{Key: proto.String("node"), Value: proto.String(node)},
		{Key: proto.String("category"), Value: proto.String(category)},
		{Key: proto.String("log"), Value: proto.String(content)},
	}
}
