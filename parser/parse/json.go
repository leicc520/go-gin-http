package parse

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/antchfx/jsonquery"
	"github.com/leicc520/go-gin-http"
)

type JsonParseSt struct {
	node *jsonquery.Node
}

//解析数据资料信息
func NewJsonParser(jsonStr string) (parser *JsonParseSt, err error) {
	defer func() { //数据为空的情况逻辑
		if e := recover(); e != nil {
			err = errors.New(fmt.Sprintf("%v", e))
		}
	}()
	jsonStr = strings.ReplaceAll(jsonStr, "\n", "")
	jsonStr = strings.ReplaceAll(jsonStr, "\r", "")
	jsonStr = strings.ReplaceAll(jsonStr, "\t", "")
	//替换对象key的单引号逻辑
	reg, _ := regexp.Compile(`'[^']+?'[\s]*:`)
	aStr := reg.FindAllString(jsonStr, -1)
	if aStr != nil && len(aStr) > 0 { //替换单引号的key信息
		for _, lStr := range aStr {
			nStr := strings.ReplaceAll(lStr, "'", "\"")
			jsonStr = strings.ReplaceAll(jsonStr, lStr, nStr)
		}
	}
	//针对json格式数据的过滤处理逻辑
	jsonStr = core.StripQuotes(jsonStr)
	jsonStr = regexp.MustCompile(`\s*,\s*]`).ReplaceAllString(jsonStr, "]")
	//这里主要是适配js json格式不严格的问题
	jsonStr = strings.ReplaceAll(jsonStr, ",]", "")
	topNode, err := jsonquery.Parse(strings.NewReader(jsonStr))
	if err != nil { //记录请求非json的日志信息
		core.LogActionOnce("json@"+core.Md5Str(jsonStr), 86400, jsonStr)
		return nil, err
	}
	parser = &JsonParseSt{node: topNode}
	return
}

//通过文件获取解析器的逻辑
func NewJsonParserFromFile(file string) (*JsonParseSt, error) {
	jsonStr := core.ReadFile(file)
	return NewJsonParser(jsonStr)
}

//获取节点的数据资料信息
func (s *JsonParseSt) InnerText(expr string) (text string, err error) {
	node, err := jsonquery.Query(s.node, expr)
	if err != nil {
		return
	}
	if node == nil {
		err = ErrNoExists
		return
	}
	value, ok := node.Value(), false
	if text, ok = value.(string); !ok {
		if byteStr, err := json.Marshal(value); err == nil {
			text = string(byteStr)
		}
	}
	return
}

//获取节点的数据资料信息
func (s *JsonParseSt) InnerTexts(expr string) (texts []string, err error) {
	nodes, err := jsonquery.QueryAll(s.node, expr)
	if err != nil {
		return
	}
	texts = make([]string, 0)
	for _, node := range nodes {
		texts = append(texts, strings.TrimSpace(node.InnerText()))
	}
	return
}

//验证是否取到 节点数据
func (s *JsonParseSt) HasNode(expr string) (has bool, err error) {
	node, err := jsonquery.Query(s.node, expr)
	if err == nil && node != nil {
		has = true
	}
	return
}

//验证节点取值是否true
func (s *JsonParseSt) NodeValueIsTrue(expr string) (r bool, err error) {
	node, err := jsonquery.Query(s.node, expr)
	if err != nil {
		return
	}
	if node == nil {
		err = ErrNoExists
		return
	}
	if node.InnerText() == "true" {
		r = true
	} else {
		r = false
	}
	return
}
