package parse

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"git.ziniao.com/webscraper/go-gin-http"
)

var (
	IsDebug = false

	ErrNoExists = errors.New("要解析的节点数据不存在!")
	ErrType     = errors.New("类型不支持,无法操作!")
	ErrNode     = errors.New("节点配置异常,无法解析")
	ErrEmpty    = errors.New("提取节点内容为空!")
	ErrOmitting = errors.New("非必须节点,检测节点不存在")
)

type RegExpParseSt string

// 根据提取规格获取数据
type IFNodeParser interface {
	InnerText(expr string) (text string, err error)
	InnerTexts(expr string) (texts []string, err error)
}

// 编译器的实现接口类型 解决依赖问题
type IFCompiler interface {
	SetDoc(doc string)
	GetDoc() string
	Clone(doc string) IFCompiler
	GetError() *ParseError
	GetParser(nodeType int8) IFNodeParser
}

// 将结果转换成slice
func convertSlice(result interface{}) []string {
	if aStr, ok := result.([]string); ok {
		return aStr
	}
	aStr := []string{fmt.Sprintf("%v", result)}
	return aStr
}

// 转换成字符串
func convertString(result interface{}) string {
	if aStr, ok := result.([]string); ok {
		return strings.Join(aStr, "\r\n")
	}
	return fmt.Sprintf("%v", result)
}

// 判定解析的内容是否为空的情况
func isEmptyResult(result interface{}) bool {
	if aStr, ok := result.([]string); ok && len(aStr) < 1 {
		return true
	}
	if str, ok := result.(string); ok && len(str) < 1 {
		return true
	}
	return false
}

// 过滤标签处理逻辑
func stripTags(result interface{}) interface{} {
	if lStr, ok := result.(string); ok {
		return core.StripTags(lStr)
	}
	if aStr, ok := result.([]string); ok {
		for idx, lStr := range aStr {
			aStr[idx] = core.StripTags(lStr)
		}
		return aStr
	}
	return result
}

// 验证结果界面是否为空的处理逻辑
func isEmpty(result interface{}) bool {
	if lStr, ok := result.(string); ok {
		return len(strings.TrimSpace(lStr)) == 0
	}
	if aStr, ok := result.([]string); ok {
		return len(aStr) == 0
	}
	return false
}

// 获取节点的数据资料信息
func (s RegExpParseSt) InnerText(expr string) (text string, err error) {
	var reg *regexp.Regexp
	reg, err = regexp.Compile(expr)
	if err != nil {
		return
	}
	text = reg.FindString(string(s))
	if len(text) < 1 {
		err = ErrNoExists
	}
	return
}

// 获取节点的数据资料信息
func (s RegExpParseSt) InnerTexts(expr string) (texts []string, err error) {
	var reg *regexp.Regexp
	reg, err = regexp.Compile(expr)
	if err != nil {
		return
	}
	texts = reg.FindAllString(string(s), -1)
	if len(texts) < 1 {
		err = ErrNoExists
	}
	return
}
