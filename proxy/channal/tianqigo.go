package channal

import (
	"errors"
	"github.com/leicc520/go-orm/log"
	"io"
	"net/http"
	"strings"
)

type SkyGoSt struct {
	BaseProxySt
}

func skyGoProxy(proto string, proxy IFProxy) error {
	sp, err := (&http.Client{}).Get(SKY_GO_PROXY + "/getip?" + proxy.GetParam() + "&_" + proto)
	if err != nil || sp == nil || sp.StatusCode != http.StatusOK {
		log.Write(-1, PROXY_CHANNEL_SKYGO, "proxy error ", err)
		return err
	}
	defer sp.Body.Close()
	body, _ := io.ReadAll(sp.Body)
	bodyStr := strings.TrimSpace(string(body))
	ipList := strings.Split(bodyStr, "\n")
	if len(ipList) < 1 || strings.Contains(bodyStr, "html") {
		log.Write(-1, PROXY_CHANNEL_SKYGO, "代理请求异常", bodyStr)
		return errors.New("代理请求获取地址异常")
	}
	proxy.SetIP(ipList) //更新IP列表
	return nil
}

func init() { //注册到注册器当中
	s := &SkyGoSt{}
	s.init(PROXY_CHANNEL_SKYGO, skyGoProxy)
	proxyRegister(PROXY_CHANNEL_SKYGO, s)
}
