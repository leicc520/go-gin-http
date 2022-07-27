package core

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/gin-gonic/gin"
	"github.com/leicc520/go-orm/log"
)

type HttpError struct {
	Code int     `json:"code"`
	Msg string   `json:"msg"`
	Debug string `json:"debug"`
}

func (self *HttpError) Error() string {
	return self.Msg
}

//出错的模式情况
func (self *HttpError) SetDebug(msg string) *HttpError {
	self.Debug = msg
	return self
}

func (self *HttpError) ToMap() map[string]interface{} {
	data := map[string]interface{}{"code":self.Code, "debug":self.Debug, "msg":self.Msg}
	return data
}

type HttpView struct {
	HttpError
	ctx *gin.Context  `json:"-"`
	Datas interface{} `json:"datas"`
}

//new 创建一个对外的执行示例
func NewHttpView(ctx *gin.Context) *HttpView {
	view := &HttpView{ctx: ctx}
	view.Msg, view.Code = "OK", 0
	return view
}


func DecryptBind(c *gin.Context, obj interface{}) error {
	sKey := c.GetHeader(EncryptKeys)
	if len(sKey) > 6 {//解密的数据业务处理逻辑
		cryptSt := &Crypt{JKey: []byte(sKey)}
		c.Set(EncryptName, cryptSt) //设置数据解码
		oldStr, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Write(log.ERROR, "请求数据解码获取数据的时候异常", err)
			return err
		}
		log.Write(log.INFO, "数据接收:", string(oldStr))
		defer c.Request.Body.Close()
		req := cryptSt.Decrypt(oldStr)
		log.Write(log.INFO, "数据解码:", string(req))
		if req == nil || len(req) < 1 {
			log.Write(log.ERROR, "数据解码:", string(oldStr))
			return errors.New("数据解码失败,无法操作.")
		}
		if err = json.Unmarshal(req, obj); err != nil {
			log.Write(log.ERROR, "数据解码:", string(req), err)
			return err
		}
		if err = ValidateStruct(obj); err != nil {
			log.Write(log.ERROR, "结构校验:", string(req), err)
			return err
		}
	} else {//非加密的业务处理逻辑
		if err := c.ShouldBind(&obj); err != nil {
			return  err
		}
	}
	return nil
}

//数据的加密处理逻辑
func (c *HttpView) enCrypt() {
	if objSt, ok := c.ctx.Get(EncryptName); ok {
		bStr, _ := json.Marshal(c)
		cryptSt := objSt.(*Crypt)
		if str, err := cryptSt.Encrypt(bStr); err == nil {
			c.ctx.String(200, str)
		}
	}
	c.ctx.JSON(200, c)
}

//出错的模式情况
func (c *HttpView)ErrorDisplay(code int, msg string) {
	c.Msg, c.Code = msg, code
	c.enCrypt()
}

//json 数据模式格式化输出
func (c *HttpView)JsonDisplay(datas interface{}) {
	c.Datas = datas
	c.enCrypt()
}
