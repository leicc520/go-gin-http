package channal

import (
	"errors"
	"git.ziniao.com/webscraper/go-orm/log"
	"io"
	"net/http"
	"strings"
)

// https://www.ipipgo.com/extractApi
type IPIPGoSt struct {
	BaseProxySt
}

// cty=US&c=100&pt=1&ft=txt&pat=\n&rep=1&key=xxxx&ts=3
func ipipGoProxy(proto string, proxy IFProxy) error {
	sp, err := (&http.Client{}).Get(IPIP_GO_PROXY + "/ip?" + proxy.GetParam() + "&_" + proto)
	if err != nil || sp == nil || sp.StatusCode != http.StatusOK {
		log.Write(-1, PROXY_CHANNEL_IPIPGO, "proxy error ", err)
		return err
	}
	defer sp.Body.Close()
	body, _ := io.ReadAll(sp.Body)
	bodyStr := strings.TrimSpace(string(body))
	ipList := strings.Split(bodyStr, "\n")
	if len(ipList) < 1 || strings.Contains(bodyStr, "html") {
		log.Write(-1, PROXY_CHANNEL_IPIPGO, "代理请求异常", bodyStr)
		return errors.New("代理请求获取地址异常")
	}
	proxy.SetIP(ipList) //更新IP列表
	return nil
}

func init() { //注册到注册器当中
	s := &IPIPGoSt{}
	s.init(PROXY_CHANNEL_IPIPGO, ipipGoProxy)
	proxyRegister(PROXY_CHANNEL_IPIPGO, s)
}
