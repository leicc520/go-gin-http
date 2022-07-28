package core

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leicc520/go-orm"
	"testing"
)

func TestAPP(t *testing.T) {
	config := AppConfigSt{Host: "127.0.0.1:8081", Name: "go.test.srv", Domain: "127.0.0.1:8081"}
	jaeger := JaegerTracingConfigSt{
		Agent: "127.0.0.1:6831",
		Type: "const",
		Param: 1,
		IsTrace: true,
	}
	jaeger.Init("go.test.srv")


	NewApp(&config).RegHandler(func(c *gin.Engine) {
		c.GET("/demo", func(context *gin.Context) {
			context.JSON(200, orm.SqlMap{"demo":"test"})
		})
		c.POST("/demov2", func(context *gin.Context) {
			args := struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{}
			if err := ShouldBind(context, &args); err != nil {
				PanicValidateHttpError(1001, err)
			}
			context.JSON(200, args)
		})
	}).Start()
}

func TestView(t *testing.T) {
	view := &HttpView{}
	view.Code = 500
	view.Msg  = "demo"
	view.Datas= "demo111"

	str, err := json.Marshal(view)
	fmt.Println(string(str), err)
}

func TestCrypt(t *testing.T) {
	sKey := "simlife@123"
	cryptSt := Crypt{JKey: []byte(sKey)}
	oldStr := "{\"name\":\"leicc\",\"age\":15}"
	newStr, err := cryptSt.Encrypt([]byte(oldStr))
	fmt.Println(newStr, err)
	unpackStr := cryptSt.Decrypt([]byte(newStr))
	fmt.Println(string(unpackStr))
}

func TestHttpRequest(t *testing.T) {
	req := NewHttpRequest()
	sKey := "simlife@123"
	cryptSt := Crypt{JKey: []byte(sKey)}
	oldStr := "{\"name\":\"leicc\",\"age\":15.12}"
	newStr, err := cryptSt.Encrypt([]byte(oldStr))
	fmt.Println(newStr, err)
	url := "http://127.0.0.1:8081/demov2"
	result := req.AddHeader(EncryptKeys, sKey).Request(url, []byte(newStr), "POST")
	if len(result) > 0 {
		ostr := cryptSt.Decrypt(result)
		fmt.Println(string(ostr), "===============")
	}
}