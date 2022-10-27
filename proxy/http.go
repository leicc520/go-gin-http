package proxy

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.ziniao.com/webscraper/go-orm"
	"git.ziniao.com/webscraper/go-orm/log"
	"golang.org/x/net/http2"
)

const CONTENT_TYPE = "content-type"

type HttpSt struct {
	sp           *http.Response
	query        url.Values
	isRedirect   bool
	monitor      *Monitor
	proxy        string
	proxyIndex   int  //代理索引
	isValid      bool //是否验证
	isProxy      bool //是否开启自动代理
	timeout      time.Duration
	cookieJar    *cookiejar.Jar
	tlsTransport http.RoundTripper
	header       map[string]string
}

// 设置结果处理检测回调验证逻辑，特殊域名的逻辑检测
type ResultCheckHandle func(link, result string, s *HttpSt) (int, error)

var (
	resultCheck ResultCheckHandle = nil
)

func CancelRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

// 设置结果回调检测句柄
func SetResultCheck(handle ResultCheckHandle) {
	resultCheck = handle
}

func NewHttpRequest() *HttpSt {
	cookieJar, _ := cookiejar.New(nil)
	return &HttpSt{query: url.Values{}, isRedirect: true, header: nil, proxyIndex: -1,
		timeout: 60 * time.Second, cookieJar: cookieJar}
}

// 设置其完成打码逻辑
func (s *HttpSt) SetIsValid(valid bool) {
	s.isValid = valid
}

// 获取大码状态
func (s *HttpSt) IsValid() bool {
	return s.isValid
}

// 设置代理监控，从这里获得代理
func (s *HttpSt) SetMonitor(monitor *Monitor) *HttpSt {
	s.monitor = monitor
	return s
}

// 获取代理的设置状态信息
func (s *HttpSt) ProxyState() bool {
	if s.isProxy && s.monitor != nil {
		return true
	}
	return false
}

// 设置请求的header业务数据信息
func (s *HttpSt) SetTimeout(timeout int) *HttpSt {
	s.timeout = time.Duration(timeout) * time.Second
	return s
}

// 设置请求的header业务数据信息
func (s *HttpSt) SetHeader(header map[string]string) *HttpSt {
	if s.header == nil { //数据为空的情况
		s.header = make(map[string]string)
	}
	for key, val := range header {
		s.header[key] = val
	}
	return s
}

// 初始化请求数据头部信息
func (s *HttpSt) initHeader(req *http.Request) *HttpSt {
	baseUrl := fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host)
	s.AddHeader("origin", baseUrl)
	if _, ok := s.header["accept"]; !ok { //不存在的话设置一下
		s.AddHeader("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	}
	if _, ok := s.header["accept-encoding"]; !ok { //不存在的话设置一下
		s.AddHeader("accept-encoding", "gzip, deflate, br")
	}
	if _, ok := s.header["upgrade-insecure-requests"]; !ok { //不存在的话设置一下
		s.AddHeader("upgrade-insecure-requests", "1")
	}
	if _, ok := s.header["sec-fetch-dest"]; !ok { //不存在的话设置一下
		s.AddHeader("sec-fetch-dest", "document")
	}
	if _, ok := s.header["sec-fetch-mode"]; !ok { //不存在的话设置一下
		s.AddHeader("sec-fetch-mode", "navigate")
	}
	if _, ok := s.header["sec-fetch-site"]; !ok { //不存在的话设置一下
		s.AddHeader("sec-fetch-site", "none")
	}
	if _, ok := s.header["sec-fetch-user"]; !ok { //不存在的话设置一下
		s.AddHeader("sec-fetch-user", "?1")
	}
	if _, ok := s.header["accept-language"]; !ok { //不存在的话设置一下
		s.AddHeader("accept-language", "*")
	}
	if _, ok := s.header["referer"]; !ok { //不存在的话设置一下
		s.AddHeader("referer", baseUrl+"/")
	}
	if _, ok := s.header["user-agent"]; !ok { //不存在的话设置一下
		s.AddHeader("user-agent", UserAgent(DEVICE_PC))
	}
	if s.header != nil && len(s.header) > 0 {
		for key, val := range s.header {
			req.Header.Set(key, val)
		}
	}
	return s
}

// 重置浏览器agent数据
func (s *HttpSt) ResetAgent(device string) string {
	agent := UserAgent(device)
	s.AddHeader("user-agent", agent)
	return agent
}

// 设置是否重定向处理逻辑,默认true
func (s *HttpSt) IsRedirect(is bool) *HttpSt {
	s.isRedirect = is
	return s
}

// 获取指定的cookie信息
func (s *HttpSt) GetJarCookie(link, name string) string {
	u, _ := url.Parse(link)
	cookies := s.cookieJar.Cookies(u)
	for _, item := range cookies {
		if item.Name == name {
			return item.Value
		}
	}
	return ""
}

// 获取所有的cookie信息
func (s *HttpSt) GetAllCookie(link string) string {
	u, _ := url.Parse(link)
	arrStr := make([]string, 0)
	cookies := s.cookieJar.Cookies(u)
	for _, item := range cookies {
		arrStr = append(arrStr, item.Name+"="+item.Value)
	}
	return strings.Join(arrStr, ";")
}

// 返回数据记录信息
func (s *HttpSt) GetResponse() *http.Response {
	return s.sp
}

// 设置请求的header业务数据信息
func (s *HttpSt) AddHeader(key, val string) *HttpSt {
	if s.header == nil {
		s.header = make(map[string]string, 0)
	}
	s.header[key] = val
	return s
}

// 设置请求的header业务数据信息
func (s *HttpSt) DelHeader(header map[string]string) *HttpSt {
	if s.header != nil {
		for key, _ := range header {
			if _, ok := s.header[key]; ok {
				delete(s.header, key)
			}
		}
	}
	return s
}

// 装入cookie数据资料信息
func (s *HttpSt) SetCookie(rawUrl, cookieStr string, isExpire bool) {
	u, _ := url.Parse(rawUrl)
	cookie := make([]*http.Cookie, 0)
	astr := strings.Split(cookieStr, ";")
	expire := time.Now().Add(time.Hour * 24 * 30)
	if isExpire { //设置过期的话清除cookie
		expire = time.Now().Add(0 - time.Hour*24*30)
	}
	for _, str := range astr {
		str = strings.TrimSpace(str)
		if len(str) < 1 {
			continue
		}
		value := strings.Split(str, "=")
		if len(value) == 2 {
			cookie = append(cookie, &http.Cookie{Name: value[0], Value: value[1], Domain: "." + u.Host, Path: "/", Expires: expire})
		}
	}
	s.cookieJar.SetCookies(u, cookie)
}

// 设置发起json的业务请求json,xml,default
func (s *HttpSt) SetContentType(typeStr string) *HttpSt {
	if s.header == nil {
		s.header = map[string]string{}
	}
	if len(typeStr) < 1 { //删除逻辑
		delete(s.header, CONTENT_TYPE)
		return s
	}
	switch strings.ToLower(typeStr) {
	case "json":
		s.header[CONTENT_TYPE] = "application/json; charset=utf-8"
	case "xml":
		s.header[CONTENT_TYPE] = "application/xml; charset=utf-8"
	default:
		s.header[CONTENT_TYPE] = "application/x-www-form-urlencoded; charset=utf-8"
	}
	return s
}

// 添加设置查询语句
func (s *HttpSt) Set(name, value string) *HttpSt {
	s.query.Set(name, value)
	return s
}

// 获取设置的http请求header数据
func (s *HttpSt) Header() map[string]string {
	return s.header
}

// 获取查询的语句数据
func (s *HttpSt) Query() string {
	return s.query.Encode()
}

// 重置请求的参数数据信息
func (s *HttpSt) Reset() *HttpSt {
	s.query = url.Values{}
	s.header = nil
	s.cookieJar, _ = cookiejar.New(nil)
	return s
}

// 重置请求的参数数据信息
func (s *HttpSt) SetTls(keySsl, pemSsl string) *HttpSt {
	c, _ := tls.X509KeyPair([]byte(pemSsl), []byte(keySsl))
	cfg := &tls.Config{
		Certificates: []tls.Certificate{c},
	}
	s.tlsTransport = http.RoundTripper(&http.Transport{
		TLSClientConfig: cfg,
	})
	return s
}

// 重置请求的参数数据信息
func (s *HttpSt) SetTlsV2(pemSsl string) *HttpSt {
	caCert, err := ioutil.ReadFile(pemSsl)
	if err != nil {
		log.Write(log.ERROR, "SetTlsV2", err)
		return s
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}
	s.tlsTransport = http.RoundTripper(&http2.Transport{
		TLSClientConfig: tlsConfig,
	})
	return s
}

// 设置是否自动代理处理逻辑
func (s *HttpSt) IsProxy(isProxy bool) *HttpSt {
	s.isProxy = isProxy
	return s
}

// 重置代理处理逻辑 -关闭代理
func (s *HttpSt) CloseProxy() {
	s.proxyIndex, s.proxy = -1, ""
	s.tlsTransport = nil
}

// 代理池中自动获取代理
func (s *HttpSt) autoSetProxy() {
	//开启了自动代理，且未设置代理的情况
	if s.isProxy && len(s.proxy) < 1 && s.monitor != nil {
		s.proxyIndex, s.proxy = s.monitor.Proxy()
		if s.proxyIndex >= 0 && len(s.proxy) > 0 {
			s.Proxy(s.proxy)
			log.Write(log.INFO, "设置代理请求:"+s.proxy)
		}
	}
}

// 设置启动http代理发起业务请求
func (s *HttpSt) Proxy(proxyUrl string) *HttpSt {
	uri, err := url.Parse(proxyUrl)
	if err != nil {
		log.Write(log.ERROR, "set proxy error", proxyUrl, err)
		return s
	}
	s.proxy = proxyUrl
	t := &http.Transport{TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	}, TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)}
	t.Proxy = http.ProxyURL(uri)
	s.tlsTransport = http.RoundTripper(t)
	return s
}

// 获取代理数据资料信息
func (s *HttpSt) GetProxy() string {
	return s.proxy
}

// 上传文件处理逻辑 封装成byte
func (s *HttpSt) UpFile(param map[string]string, paramName, path, fileName string) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if len(path) > 0 && orm.FileExists(path) {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		if fileName == "" {
			fileName = filepath.Base(path)
		}
		fp, err := writer.CreateFormFile(paramName, fileName)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(fp, file)
		if err != nil {
			return nil, err
		}
	}
	for key, val := range param {
		_ = writer.WriteField(key, val)
	}
	s.SetHeader(map[string]string{"content-type": writer.FormDataContentType()})
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

// 请求下载文件数据信息
func (s *HttpSt) DownLoad(url, filePath string) (string, error) {
	var fp *os.File = nil
	var sp *http.Response = nil
	defer func() { //补货异常的处理逻辑
		if sp != nil && sp.Body != nil {
			sp.Body.Close()
		}
		if r := recover(); r != nil {
			log.Write(log.ERROR, "request url ", url, "error", r)
		}
		if fp != nil {
			fp.Close()
		}
	}()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Write(log.ERROR, url, err)
		return "", err
	}
	if s.header != nil && len(s.header) > 0 {
		for key, val := range s.header {
			req.Header.Set(key, val)
		}
	}
	client := &http.Client{Timeout: s.timeout, Jar: s.cookieJar}
	if s.tlsTransport != nil { //设置加密请求业务逻辑
		client.Transport = s.tlsTransport
	}
	if sp, err = client.Do(req); err != nil || sp == nil {
		log.Write(log.ERROR, url, err)
		return "", err
	}
	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	io.Copy(f, sp.Body)
	return filePath, nil
}

// 请求下载文件数据信息
func (s *HttpSt) Stream(url string) ([]byte, error) {
	var sp *http.Response = nil
	defer func() { //补货异常的处理逻辑
		if sp != nil && sp.Body != nil {
			sp.Body.Close()
		}
		if r := recover(); r != nil {
			log.Write(log.ERROR, "request url ", url, "error", r)
		}
	}()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Write(log.ERROR, url, err)
		return nil, err
	}
	if s.header != nil && len(s.header) > 0 {
		for key, val := range s.header {
			req.Header.Set(key, val)
		}
	}
	client := &http.Client{Timeout: s.timeout, Jar: s.cookieJar}
	if s.tlsTransport != nil { //设置加密请求业务逻辑
		client.Transport = s.tlsTransport
	}
	if sp, err = client.Do(req); err != nil || sp == nil {
		log.Write(log.ERROR, url, err)
		return nil, err
	}
	return io.ReadAll(sp.Body)
}

// 发起一个http业务请求
func (s *HttpSt) Request(url string, body []byte, method string) (result string, err error) {
	s.sp, s.isValid = nil, false
	defer func() { //捕获异常的处理逻辑
		if s.sp != nil && s.sp.Body != nil {
			s.sp.Body.Close()
		}
		if r := recover(); r != nil {
			log.Write(log.ERROR, "request url ", url, "error", r)
		}
	}()
	//针对GET请求的处理参数
	if method == http.MethodGet && len(body) > 0 {
		if strings.Contains(url, "?") {
			url += "&" + string(body)
		} else {
			url += "?" + string(body)
		}
		body = nil
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		log.Write(log.ERROR, url, err, string(body))
		return
	}
	s.initHeader(req) //初始化附加头信息
	s.autoSetProxy()  //判定并设置自动代理
	client := &http.Client{Timeout: s.timeout, Jar: s.cookieJar}
	if s.tlsTransport != nil { //设置加密请求业务逻辑
		client.Transport = s.tlsTransport
	}
	//关闭重定向处理逻辑
	if !s.isRedirect {
		client.CheckRedirect = CancelRedirect
	}
	s.sp, err = client.Do(req)
	statusCode := -1
	if s.sp != nil {
		statusCode = s.sp.StatusCode
	}
	if s.monitor != nil { //如果设置了监控逻辑
		s.monitor.Report(s.proxyIndex, req.Host, statusCode)
	}
	if statusCode == 404 { //页面不存在的情况
		return "", ErrHttp404NotFound
	}
	if err != nil || s.sp == nil {
		s.CloseProxy() //失败的情况关闭
		log.Write(-1, url, err, s.proxy, string(body))
		return
	}
	if s.sp.StatusCode == http.StatusOK {
		return s.readResult(url, req.Host) //返回请求回来的数据信息
	} else {
		if s.sp != nil && s.sp.Body != nil {
			result, _ = s.readResult(url, req.Host) //返回请求回来的数据信息
		}
		log.Write(-1, url, s.sp.Status, s.proxy, result)
		s.CloseProxy() //失败的情况关闭
		return result, errors.New(s.sp.Status)
	}
}

// 检测编码业务处理逻辑
func (s *HttpSt) isIso88591() bool {
	if s.sp == nil || s.sp.Header == nil {
		return false
	}
	return strings.Contains(s.sp.Header.Get("Content-Type"), "charset=iso-8859-1")
}

// 读取解码的数据资料信息
func (s *HttpSt) readResult(url, host string) (result string, err error) {
	var body []byte
	if s.sp.Header.Get("Content-Encoding") == "gzip" {
		body, err = GZIPDecode(s.sp.Body)
	} else {
		body, err = ioutil.ReadAll(s.sp.Body)
	}
	if err != nil {
		log.Write(-1, "请求读取结果异常", s.sp, err)
		return
	}
	if s.isIso88591() && len(body) > 0 {
		result, err = Decode("iso-8859-1", body)
	} else {
		result = string(body)
	}
	//针对结果的检测处理逻辑
	if resultCheck != nil { //检测结果
		statusCode, pageErr := resultCheck(url, result, s)
		if pageErr != nil {
			err = pageErr
			if s.monitor != nil { //如果设置了监控逻辑
				s.monitor.Report(s.proxyIndex, host, statusCode)
			}
		}
	}
	//过滤一下空格等等数据
	result = normalizeSpace(result)
	return
}
