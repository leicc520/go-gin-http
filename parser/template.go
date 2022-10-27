package parser

import (
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/leicc520/go-gin-http"
	"github.com/leicc520/go-gin-http/parser/parse"
	"github.com/leicc520/go-orm/log"
	"gopkg.in/yaml.v2"
)

/*
********************************************************************
配置模板参数数据资料信息，只有user-agent是随机的，其他走配置
*/
type TemplateSt struct {
	Request    *BaseRequest      `json:"request" yaml:"request"`
	DataFields []parse.ElementSt `json:"dataFields" yaml:"dataFields"`
}

// 统计必填的字段数据 只要一级的必填字段即可
func (s *TemplateSt) RequiredElementStats() int {
	requireStat := 0
	for _, item := range s.DataFields {
		if item.IsOmitting == 0 {
			requireStat += 1
		}
	}
	return requireStat
}

// 检测必填的字段是否全部覆盖完整
func (s *TemplateSt) IsAllCollected(result map[string]interface{}) (bool, int, error) {
	var err = errors.New("Empty")
	if result == nil || len(result) < 1 {
		return false, len(s.DataFields), err
	}
	//只检测一级节点是否都完整
	flag, aStr, nError := true, []string{}, 0
	for _, item := range s.DataFields {
		if _, ok := result[item.Tag]; ok {
			continue
		}
		aStr = append(aStr, item.Tag)
		if item.IsOmitting == 0 { //必填字段不存在
			nError++
			flag = false
		}
	}
	if flag { //完全匹配的处理逻辑
		return flag, 0, nil
	}
	return flag, nError, errors.New(strings.Join(aStr, ";"))
}

// 加载配置数据资料信息
func (s *TemplateSt) LoadFile(confFile string) error {
	if file, err := os.Stat(confFile); err != nil || !file.Mode().IsRegular() {
		log.Write(log.ERROR, "load Template Config File Failed: ", err)
		return err
	}
	data, _ := os.ReadFile(confFile)
	//把yaml形式的字符串解析成struct类型 先子类初始化
	if err := yaml.Unmarshal(data, s); err != nil {
		log.Write(log.ERROR, "load Template Config Parse Failed: ", err)
		return err
	}
	return nil
}

// 抓取网页数据处理逻辑
func (s *TemplateSt) Crawling(url string) string {
	doc, _, err := s.Request.Do(nil, url, SpiderDataExpire)
	if err != nil {
		return ""
	}
	return doc
}

// 配置模板数据资料信息
type TemplatesSt struct {
	l         sync.Mutex
	templates map[string]*TemplateSt
}

// 初始化逻辑
func NewTemplates() *TemplatesSt {
	return &TemplatesSt{templates: make(map[string]*TemplateSt), l: sync.Mutex{}}
}

// 获取数据资料信息
func (s *TemplatesSt) Get(file string) *TemplateSt {
	s.l.Lock()
	defer s.l.Unlock()
	md5Str := core.Md5Str(file)
	if tpl, ok := s.templates[md5Str]; ok {
		return tpl
	}
	tpl := &TemplateSt{Request: &BaseRequest{}}
	if err := tpl.LoadFile(file); err != nil {
		return nil
	}
	s.templates[md5Str] = tpl
	return tpl
}
