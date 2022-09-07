package test

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leicc520/go-gin-http"
	"github.com/leicc520/go-gin-http/micro"
	"github.com/leicc520/go-orm"
	"testing"
)

func TestAPP(t *testing.T) {
	defer func() {
		fmt.Println("-----------------")
	}()
	micro.CmdInit(func() {
	//	core.SetRegSrv(micro.NewRegSrvClient)
	}) //初始化配置
	/*
	jaeger := tracing.JaegerTracingConfigSt{
		Agent: "127.0.0.1:6831",
		Type: "const",
		Param: 1,
		IsTrace: true,
	}
	core.InjectTracing(&jaeger)*/
	config := core.AppConfigSt{Host: "127.0.0.1:8081", Name: "go.demov5.srv", Version: "v1.0.0", Domain: "127.0.0.1:8081"}
	core.NewApp(&config).RegHandler(func(c *gin.Engine) {
		c.GET("/demo", func(context *gin.Context) {
			context.JSON(200, orm.SqlMap{"demo":"test"})
		})
		c.POST("/demov2", func(context *gin.Context) {
			args := struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{}
			if err := core.ShouldBind(context, &args); err != nil {
				core.PanicValidateHttpError(1001, err)
			}
			core.NewHttpView(context).JsonDisplay(args)
		})
		c.GET("/test", func(context *gin.Context) {
			req := core.NewHttpRequest().InjectTrace(context)
			sKey := "simlife@123"
			cryptSt := core.Crypt{JKey: []byte(sKey)}
			oldStr := "{\"name\":\"leicc\",\"age\":15}"
			newStr, err := cryptSt.Encrypt([]byte(oldStr))
			fmt.Println(newStr, err)
			url := "http://127.0.0.1:8081/demov2"
			result := req.AddHeader(core.EncryptKeys, sKey).Request(url, []byte(newStr), "POST")
			var ostr []byte = nil
			if len(result) > 0 {
				ostr = cryptSt.Decrypt(result)
				fmt.Println(string(ostr), "===============")
			}
			urlv2 := "http://127.0.0.1:8081/demo"
			result = req.Reset().Request(urlv2, nil, "GET")
			fmt.Println(string(result), "===============")
			context.JSON(200, orm.SqlMap{"demov2":string(ostr), "demo":string(result)})
		})
	}).Start()
}

func TestView(t *testing.T) {
	view := &core.HttpView{}
	view.Code = 500
	view.Msg  = "demo"
	view.Datas= "demo111"
	str, err := json.Marshal(view)
	fmt.Println(string(str), err)
}

func TestCrypt(t *testing.T) {
	sKey := "simlife@123"
	cryptSt := core.Crypt{JKey: []byte(sKey)}
	oldStr := "{\"name\":\"leicc\",\"age\":15}"
	newStr, err := cryptSt.Encrypt([]byte(oldStr))
	fmt.Println(newStr, err)
	unpackStr := cryptSt.Decrypt([]byte(newStr))
	fmt.Println(string(unpackStr))
}

func TestHttpRequest(t *testing.T) {
	req := core.NewHttpRequest()
	sKey := "simlife@123"
	cryptSt := core.Crypt{JKey: []byte(sKey)}
	oldStr := "{\"name\":\"leicc\",\"age\":15}"
	newStr, err := cryptSt.Encrypt([]byte(oldStr))
	fmt.Println(newStr, err)
	url := "http://127.0.0.1:8081/demov2"
	result := req.AddHeader(core.EncryptKeys, sKey).Request(url, []byte(newStr), "POST")
	if len(result) > 0 {
		ostr := cryptSt.Decrypt(result)
		fmt.Println(string(ostr), "===============")
	}
}