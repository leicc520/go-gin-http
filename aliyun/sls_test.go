package aliyun

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestSlsLog(t *testing.T) {
	slsConfig := AliYunSLSSt{EndPint: os.Getenv("SLSEndpoint"), SLSTopic: os.Getenv("SLSTopic"), SLSProject: os.Getenv("SLSProject"),
		AccessKeyID: os.Getenv("SLSAccessKeyID"), AccessKeySecret: os.Getenv("SLSAccessKeySecret"), SLSStoreBucket: os.Getenv("SLSStoreBucket")}
	slsConfig.InitProducer()
	fmt.Printf("%+v", slsConfig)
	log := SLSWrapped("demo", "this is a demo", "local")
	SendSLSLog(log)
	time.Sleep(time.Second * 3)

}
