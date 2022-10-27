package core

import (
	"bytes"
	"crypto/tls"
	"github.com/opentracing/opentracing-go"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"git.ziniao.com/webscraper/go-gin-http/tracing"
	"git.ziniao.com/webscraper/go-orm/log"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go/ext"
)

const CONTENT_TYPE = "content-type"

// 获取查询的语句数据
func keySort(data map[string]interface{}) []string {
	keys := make([]string, 0)
	for key, _ := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type HttpSt struct {
	sp           *http.Response
	query        url.Values
	timeout      time.Duration
	cookieJar    *cookiejar.Jar
	tlsTransport *http.Transport
	tracingFunc  func(r *http.Request) opentracing.Span
	header       map[string]string
}

func NewHttpRequest() *HttpSt {
	cookieJar, _ := cookiejar.New(nil)
	return &HttpSt{query: url.Values{}, header: nil, timeout: 120 * time.Second, cookieJar: cookieJar}
}

// 设置请求的header业务数据信息
func (s *HttpSt) SetTimeout(timeout int) *HttpSt {
	s.timeout = time.Duration(timeout) * time.Second
	return s
}

// 设置请求的header业务数据信息
func (s *HttpSt) SetHeader(header map[string]string) *HttpSt {
	if s.header != nil { //数据不为空的情况
		for key, val := range header {
			s.header[key] = val
		}
	} else {
		s.header = header
	}
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

// 返回数据记录信息
func (s *HttpSt) GetResponse() *http.Response {
	return s.sp
}

// 设置请求的header业务数据信息
func (s *HttpSt) AddHeader(key, val string) *HttpSt {
	if s.header == nil {
		s.header = map[string]string{}
	}
	s.header[key] = val
	return s
}

// 设置发起json的业务请求json,xml,default
func (s *HttpSt) SetContentType(typeStr string) *HttpSt {
	if s.header == nil {
		s.header = map[string]string{}
	}
	switch strings.ToLower(typeStr) {
	case "json":
		s.header[CONTENT_TYPE] = "application/json; charset=utf-8"
	case "xml":
		s.header[CONTENT_TYPE] = "application/xml; charset=utf-8"
	default:
		s.header[CONTENT_TYPE] = "application/x-www-form-urlencoded"
	}
	return s
}

// 注入链路跟踪处理逻辑
func (s *HttpSt) InjectTrace(c *gin.Context) *HttpSt {
	spanCtx := tracing.GetTracingCtx(c)
	s.tracingFunc = func(req *http.Request) opentracing.Span {
		if spanCtx == nil {
			return nil
		}
		span := opentracing.GlobalTracer().StartSpan("http.api",
			opentracing.ChildOf(spanCtx.(opentracing.SpanContext)),
			opentracing.Tag{Key: string(ext.Component), Value: "HTTP"},
			ext.SpanKindRPCClient)
		err := opentracing.GlobalTracer().Inject(span.Context(),
			opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
		if err != nil { //异常的情况处理逻辑
			log.Write(log.ERROR, "tracing inject error", err)
		}
		return span
	}
	return s
}

// 添加设置查询语句
func (s *HttpSt) Set(name, value string) *HttpSt {
	s.query.Set(name, value)
	return s
}

// 获取查询的语句数据
func (s *HttpSt) Query() string {
	return s.query.Encode()
}

// 重置请求的参数数据信息
func (s *HttpSt) Reset() *HttpSt {
	s.query = url.Values{}
	s.header = nil
	return s
}

// 重置请求的参数数据信息
func (s *HttpSt) SetTls(keySsl, pemSsl string) *HttpSt {
	c, _ := tls.X509KeyPair([]byte(pemSsl), []byte(keySsl))
	cfg := &tls.Config{
		Certificates: []tls.Certificate{c},
	}
	s.tlsTransport = &http.Transport{
		TLSClientConfig: cfg,
	}
	return s
}

// 上传文件处理逻辑 封装成byte
func (s *HttpSt) UpFile(param map[string]string, paramName, path, fileName string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if fileName == "" {
		fileName = filepath.Base(path)
	}
	fp, err := writer.CreateFormFile(paramName, fileName)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(fp, file)
	for key, val := range param {
		_ = writer.WriteField(key, val)
	}
	s.SetHeader(map[string]string{"content-type": writer.FormDataContentType()})
	if err = writer.Close(); err != nil {
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

// 设置启动http代理发起业务请求
func (s *HttpSt) Proxy(proxyUrl string) *HttpSt {
	s.tlsTransport = &http.Transport{TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	}, TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)}
	suri, err := url.Parse(proxyUrl)
	if err == nil { // 使用传入代理
		s.tlsTransport.Proxy = http.ProxyURL(suri)
	}
	return s
}

// 发起一个http业务请求
func (s *HttpSt) Request(url string, body []byte, method string) (result []byte) {
	s.sp = nil
	defer func() { //补货异常的处理逻辑
		if s.sp != nil && s.sp.Body != nil {
			s.sp.Body.Close()
		}
		if r := recover(); r != nil {
			log.Write(log.ERROR, "request url ", url, "error", r)
			result = nil
		}
	}()
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		log.Write(log.ERROR, url, err, string(body))
		return nil
	}
	if s.header != nil && len(s.header) > 0 {
		for key, val := range s.header {
			req.Header.Set(key, val)
		}
	}
	if s.tracingFunc != nil { //链路跟踪注入处理逻辑
		span := s.tracingFunc(req)
		if span != nil { //结束链路跟踪
			defer span.Finish()
		}
	}
	client := &http.Client{Timeout: s.timeout, Jar: s.cookieJar}
	if s.tlsTransport != nil { //设置加密请求业务逻辑
		client.Transport = s.tlsTransport
	}
	s.sp, err = client.Do(req)
	if err != nil || s.sp == nil {
		log.Write(log.ERROR, url, err, string(body))
		return nil
	}
	if result, err = ioutil.ReadAll(s.sp.Body); err != nil {
		log.Write(log.ERROR, url, err, string(body))
		return nil
	} else {
		log.Write(log.INFO, url, string(result))
		return result
	}
}
