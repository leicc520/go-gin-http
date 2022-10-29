package dingding

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"git.ziniao.com/webscraper/go-gin-http"
	"git.ziniao.com/webscraper/go-orm"
)

type RobotHookSt struct {
	Link string `yaml:"link"`
	Sign string `yaml:"sign"`
	Name string `yaml:"name"`
}

type WebHookResponseSt struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

//发送机器人通知的处理逻辑
func (s *RobotHookSt) Notify(text string) WebHookResponseSt {
	client := core.NewHttpRequest()
	msgSt := orm.SqlMap{"msgtype": "text", "text": orm.SqlMap{"content": text}}
	body, _ := json.Marshal(msgSt)
	link := s.Link
	if len(s.Sign) > 0 { //如果设置的加签名
		timeStamp := time.Now().Unix() * 1000
		strToSign := fmt.Sprintf("%d\n%s", timeStamp, s.Sign)
		hash := hmac.New(sha256.New, []byte(s.Sign))
		hash.Write([]byte(strToSign))
		signData := hash.Sum(nil)
		signStr := url.QueryEscape(base64.StdEncoding.EncodeToString(signData))
		link += fmt.Sprintf("&timestamp=%d&sign=%s", timeStamp, signStr)
	}
	result := client.SetContentType("json").Request(link, body, "POST")
	if result == nil || len(result) < 1 {
		return WebHookResponseSt{Errcode: 501, Errmsg: "请求接口反馈一次,无法操作"}
	}
	data := WebHookResponseSt{}
	if err := json.Unmarshal(result, &data); err != nil {
		return WebHookResponseSt{Errcode: 502, Errmsg: err.Error()}
	}
	return data
}
