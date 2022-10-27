package channal

import (
	"errors"
	"git.ziniao.com/webscraper/go-orm/log"
	"io"
	"net/http"
	"strings"
)

type IdeaGoSt struct {
	BaseProxySt
}

func ideaGoProxy(proto string, proxy IFProxy) error {
	sp, err := (&http.Client{}).Get(IDEA_GO_PROXY + "/getProxyIp?" + proxy.GetParam() + "&_" + proto)
	if err != nil || sp == nil || sp.StatusCode != http.StatusOK {
		log.Write(-1, PROXY_CHANNEL_IDEAGO, "proxy error ", err)
		return err
	}
	defer sp.Body.Close()
	body, _ := io.ReadAll(sp.Body)
	bodyStr := strings.TrimSpace(string(body))
	ipList := strings.Split(bodyStr, "\n")
	if len(ipList) < 1 || strings.Contains(bodyStr, "html") {
		log.Write(-1, PROXY_CHANNEL_IDEAGO, "代理请求异常", bodyStr)
		return errors.New("代理请求获取地址异常")
	}
	proxy.SetIP(ipList) //更新IP列表
	return nil
}

func init() { //注册到注册器当中
	s := &IdeaGoSt{}
	s.init(PROXY_CHANNEL_IDEAGO, ideaGoProxy)
	proxyRegister(PROXY_CHANNEL_IDEAGO, s)
}
