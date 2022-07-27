package core

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
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

//数据的加密处理逻辑
func (c *HttpView)enCrypt() {
	if objSt, ok := c.ctx.Get(EncryptName); ok {
		bStr, _ := json.Marshal(c)
		cryptSt := objSt.(Crypt)
		if str, err := cryptSt.Encrypt(bStr); err == nil {
			c.ctx.String(200, str)
			return //使用加密协议返回数据
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
