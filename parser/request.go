package parser

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"git.ziniao.com/webscraper/scraper-task/lib"
	"git.ziniao.com/webscraper/scraper-task/lib/proxy"
	"github.com/leicc520/go-orm/cache"
)

const (
	SpiderDataExpire = 86400
)

var (
	monitor        *proxy.Monitor = nil
	mCache         cache.Cacher   = nil
	errUnknownPage                = errors.New("unknown page")
)

/*
*******************************************************************
请求业务的封装，获取到数据之后写缓存，然后返回
*/
type BaseRequest struct {
	Url      string            `json:"url"       yaml:"url"`
	RegUrl   []string          `json:"reg_url"   yaml:"reg_url"`
	RegMatch []string          `json:"reg_match" yaml:"reg_match"`
	Method   string            `json:"method"    yaml:"method"`
	Params   string            `json:"params"    yaml:"params"`
	Device   string            `json:"device"    yaml:"device"`
	Version  int               `json:"version"   yaml:"version"`
	Header   map[string]string `json:"headers"   yaml:"headers"`
}

// 注入缓存以及代理监控
func Inject(sCache cache.Cacher, sMonitor *proxy.Monitor) {
	if sCache != nil {
		mCache = sCache
	}
	if sMonitor != nil {
		monitor = sMonitor
	}
}

// 获取缓存策略的key
func (r *BaseRequest) CacheKey(uri *url.URL) string {
	return uri.Host + "@" + lib.Md5Str(uri.String())
}

// 通过缓存获取数据
func (r *BaseRequest) CacheGet(uri *url.URL) (ckey, result string) {
	if mCache == nil { //不做缓存处理的情况
		return
	}
	ckey = r.CacheKey(uri)
	cResult := mCache.Get(ckey)
	if cResult != nil {
		if lStr, ok := cResult.(string); ok {
			result = lStr
		}
	}
	return
}

// 验证请求的地址是否和当前任务匹配
func (r *BaseRequest) isRegUrl() bool {
	if len(r.RegUrl) > 0 {
		for _, regStr := range r.RegUrl {
			ok, err := regexp.MatchString(regStr, r.Url)
			if ok && err == nil {
				return true
			}
		}
		return false
	}
	return true
}

// 检测获取的内容是否和预期的一致
func (r *BaseRequest) isRegMatch(result string) bool {
	if len(r.RegMatch) > 0 {
		for _, regStr := range r.RegMatch { //只要检测包含想要的字段即可
			if !strings.Contains(result, regStr) {
				return false
			}
		}
	}
	return true
}

// 发起网络请求，爬取业务数据资料信息
func (r *BaseRequest) Do(client *proxy.HttpSt, link string, expire int64) (string, string, error) {
	if len(link) > 0 && strings.HasPrefix(link, "http") {
		r.Url = link
	}
	uri, err := url.Parse(r.Url)
	if err != nil {
		return "", "", errors.New("地址格式错误:" + r.Url + err.Error())
	}
	ckey, result := r.CacheGet(uri)
	if len(result) > 0 { //缓存已经存在
		return result, ckey, err
	}
	if !r.isRegUrl() { //地址不匹配的情况
		lib.LogActionOnce("url@"+lib.Md5Str(link), 86400, r.RegUrl, r.Url)
		return "", "", errors.New("地址模式不匹配:" + r.Url)
	}
	if client == nil { //不传的话 使用默认
		client = proxy.NewHttpRequest().SetMonitor(monitor)
	}
	if r.Header != nil && len(r.Header) > 0 { //设置请求头信息
		client.SetHeader(r.Header)
	}

	result, err = client.Request(r.Url, []byte(r.Params), r.Method)
	//返回的内容检测不到关键词，记录异常
	if err == nil && len(result) > 0 && !r.isRegMatch(result) {
		lib.LogActionOnce("web@"+lib.Md5Str(link), 86400, result)
		err = errUnknownPage
	}
	if err == nil && len(result) > 0 && mCache != nil { //爬取到内容了
		if expire < 1 { //默认缓存1天
			expire = SpiderDataExpire
		}
		mCache.Set(ckey, result, expire)
	}
	return result, ckey, err
}

// 清理删除缓存，如果抓取的数据不准的情况
func (r *BaseRequest) CleanCache(ckey string) {
	if mCache != nil {
		mCache.Del(ckey) //删除缓存
	}
}
