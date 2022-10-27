package channal

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/leicc520/go-orm/log"
)

type DaMaiGoSt struct {
	BaseProxySt
}

func daMaiProxy(proto string, proxy IFProxy) error {
	sp, err := (&http.Client{}).Get(DAMAI_GO_PROXY + "/index.php?" + proxy.GetParam() + "&_" + proto)
	if err != nil || sp == nil || sp.StatusCode != http.StatusOK {
		log.Write(-1, PROXY_CHANNEL_DAMAIGO, "proxy error ", err)
		return err
	}
	defer sp.Body.Close()
	body, _ := io.ReadAll(sp.Body)
	bodyStr := strings.TrimSpace(string(body))
	ipList := strings.Split(bodyStr, "\n")
	if len(ipList) < 1 || strings.Contains(bodyStr, "html") {
		log.Write(-1, PROXY_CHANNEL_DAMAIGO, "代理请求异常", bodyStr)
		return errors.New("代理请求获取地址异常")
	}
	proxy.SetIP(ipList) //更新IP列表
	return nil
}

func init() { //注册到注册器当中
	s := &DaMaiGoSt{}
	s.init(PROXY_CHANNEL_DAMAIGO, daMaiProxy)
	proxyRegister(PROXY_CHANNEL_DAMAIGO, s)
}
