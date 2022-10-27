package parse

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"git.ziniao.com/webscraper/scraper-task/lib"
	"github.com/leicc520/go-orm"
	"github.com/leicc520/go-orm/log"
)

const (
	NODE_TYPE_XPATH  = 1 //通过xpath定位元素
	NODE_TYPE_QUERY  = 2 //通过go-query查询
	NODE_TYPE_JSON   = 3 //通过json-query解析
	NODE_TYPE_REGREP = 4 //通过正则提取元素

	ELEMENT_TYPE_TEXT  = "TEXT" //采集内部元素
	ELEMENT_TYPE_HTML  = "HTML" //采集内部元素
	ELEMENT_TYPE_IMAGE = "IMG"  //采集内部图片元素
	ELEMENT_TYPE_ATTR  = "ATTR" //采集元素属性
	ELEMENT_TYPE_URL   = "URL"  //采集元素地址
	ELEMENT_TYPE_LIST  = "LIST" //采集元素列表
)

// 元素节点提取配置，便利模板节点直到知道数据才结束
type ElementSt struct {
	Tag        string      `json:"tag" yaml:"tag"`       //提取之后放到这个名字的map当中
	Name       string      `json:"name" yaml:"name"`     //元素节点名称
	Device     string      `json:"device" yaml:"device"` //移动设备还是PC
	XPath      []string    `json:"xPath" yaml:"xPath"`
	CssPath    []string    `json:"cssPath" yaml:"cssPath"`
	Json       []string    `json:"json"  yaml:"json"`
	Regexp     []string    `json:"regexp" yaml:"regexp"`
	IsOmitting int8        `json:"is_omitting" yaml:"is_omitting"`
	MatchIdx   string      `json:"matchIdx" yaml:"matchIdx"`
	MatchReg   string      `json:"matchReg" yaml:"matchReg"`
	Type       string      `json:"type" yaml:"type"`
	Children   []ElementSt `json:"children" yaml:"children"` //允许递归的获取元素，在当前解析节点继续解析
}

// 格式化成字符串输出数据
func (s ElementSt) String() string {
	arrStr := []string{s.Name, s.Tag, s.MatchReg}
	if s.XPath != nil && len(s.XPath) > 0 {
		arrStr = append(arrStr, "xpath:"+strings.Join(s.XPath, "|"))
	}
	if s.CssPath != nil && len(s.CssPath) > 0 {
		arrStr = append(arrStr, "csspath:"+strings.Join(s.CssPath, "|"))
	}
	if s.Json != nil && len(s.Json) > 0 {
		arrStr = append(arrStr, "json:"+strings.Join(s.Json, "|"))
	}
	if s.Regexp != nil && len(s.Regexp) > 0 {
		arrStr = append(arrStr, "regexp:"+strings.Join(s.Regexp, "|"))
	}
	return strings.Join(arrStr, ";")
}

// 执行业务逻辑解析处理逻辑
func (s ElementSt) RunParse(t IFCompiler, result orm.SqlMap) error {
	value, err := s.getValue(t)
	if err != nil {
		return err
	}
	isTagValue := false
	//判断是否继续匹配逻辑
	if s.Children != nil && len(s.Children) > 0 {
		aStr := convertSlice(value)
		list := make([]orm.SqlMap, 0)
		for _, doc := range aStr {
			//在每个匹配节点下接续查找数据
			newCP := t.Clone(doc)
			items := orm.SqlMap{}
			for _, el := range s.Children {
				err = el.RunParse(newCP, items)
				if err != nil {
					//针对选填字段的处理
					if err == ErrOmitting {
						t.GetError().Wrapped(el.Tag, err)
						continue
					}
					return err
				}
			}
			list = append(list, items)
		}
		isTagValue = true
		//元素数据的组织形式 对象
		if len(list) == 1 && len(list[0]) > 0 {
			result[s.Tag] = list[0]
			for key, val := range list[0] {
				if key == s.Tag && len(list[0]) == 1 {
					result[s.Tag] = val
					break
				}
			}
		} else if len(list) > 1 {
			result[s.Tag] = list
		}
	}
	//记录匹配结果到map当中 为空的字段忽略不返回
	if len(s.Tag) > 0 && !isTagValue {
		if s.Type != ELEMENT_TYPE_HTML {
			value = stripTags(value) //处理标签逻辑
		}
		if IsDebug { //调试模式
			fmt.Printf("%+v: %+v \r\n", s.Tag, value)
		}
		log.Write(log.INFO, s.Name, s.Tag, value)
		if !isEmptyResult(value) {
			result[s.Tag] = value
		}
	}
	return nil
}

// 执行获取匹配的结果数据处理逻辑
func (s ElementSt) getValue(t IFCompiler) (value interface{}, err error) {
	value, err = s.nodeParse(t)
	if err != nil { //节点取值匹配失败的情况
		if s.IsOmitting > 0 { //缺省节点，允许出错
			err = ErrOmitting
		}
		return
	}
	if len(s.MatchReg) > 0 { //正则过滤提取
		if result, err := s.regFilter(value); err == nil {
			value = result
		}
	}
	result := convertString(value)
	summary := lib.CutStr(result, 64, "...")
	log.Write(log.INFO, s.String(), summary)
	return
}

// 针对字符串内容的过滤处理逻辑
func (s ElementSt) regStrFilter(value string, reg *regexp.Regexp) (interface{}, error) {
	var err error = nil
	value = lib.StripTags(value)
	if s.MatchIdx == "-1" { //默认直接返回即可
		str := reg.FindString(value)
		if len(str) < 1 {
			err = errors.New("正则解析数据不存在(过滤器)")
		}
		return str, err
	}
	arrIdx := strings.Split(s.MatchIdx, ",")
	arrStr := reg.FindStringSubmatch(value)
	var result []string = nil
	nSize := len(arrStr)
	for _, idxStr := range arrIdx {
		idx, err := strconv.ParseInt(idxStr, 10, 64)
		if err != nil || int(idx) >= nSize {
			log.Write(-1, s.Name, s.Tag, s.MatchReg, value, "解析匹配参数异常(过滤器)")
			if err == nil { //异常的情况
				err = errors.New("匹配索引超出,无法操作(过滤器)")
			}
			return nil, err
		}
		arrStr[idx] = strings.TrimSpace(arrStr[idx])
		if len(arrIdx) == 1 { //只有一个元素直接返回
			for jdx := idx; int(jdx) < nSize; jdx++ { //查找最近一个不为空的元素
				if strings.TrimSpace(arrStr[jdx]) != "" {
					return arrStr[jdx], nil
				}
			}
			return arrStr[idx], nil
		}
		if result == nil { //为空初始化
			result = make([]string, 0)
		}
		result = append(result, arrStr[idx])
	}
	return result, nil
}

// 正则提取逻辑
func (s ElementSt) regFilter(value interface{}) (interface{}, error) {
	if reg, err := regexp.Compile(s.MatchReg); err == nil {
		if str, ok := value.(string); ok {
			return s.regStrFilter(str, reg)
		}
		if arr, ok := value.([]string); ok {
			items := make([]interface{}, 0)
			for _, str := range arr {
				if result, err := s.regStrFilter(str, reg); err != nil {
					return nil, err
				} else {
					items = append(items, result)
				}
			}
			return items, nil
		}
	}
	return nil, errors.New("过滤器未生效:" + s.MatchReg)
}

// 执行业务逻辑解析处理逻辑
func (s ElementSt) nodeParse(t IFCompiler) (result interface{}, err error) {
	if len(s.XPath) > 0 {
		p := t.GetParser(NODE_TYPE_XPATH)
		result, err = s.runParse(s.XPath, p)
		if err == nil { //模板解析提取成功
			return
		}
	}
	if len(s.CssPath) > 0 {
		p := t.GetParser(NODE_TYPE_QUERY)
		result, err = s.runParse(s.CssPath, p)
		if err == nil { //模板解析提取成功
			return
		}
	}
	if len(s.Regexp) > 0 {
		p := t.GetParser(NODE_TYPE_REGREP)
		result, err = s.runParse(s.Regexp, p)
		if err == nil { //模板解析提取成功
			return
		}
	}
	if len(s.Json) > 0 {
		p := t.GetParser(NODE_TYPE_JSON)
		result, err = s.runParse(s.Json, p)
		if err == nil { //模板解析提取成功
			return
		}
	}
	if err == nil { //节点配置异常的情况
		err = ErrNode
	}
	return
}

// 执行业务逻辑处理情况
func (s ElementSt) runParse(arrTemps []string, p IFNodeParser) (result interface{}, err error) {
	for _, tempStr := range arrTemps {
		switch s.Type {
		case ELEMENT_TYPE_TEXT:
			result, err = p.InnerText(tempStr)
		case ELEMENT_TYPE_HTML:
			result, err = p.InnerText(tempStr)
		case ELEMENT_TYPE_LIST:
			result, err = p.InnerTexts(tempStr)
		default:
			err = ErrType
			return
		}
		//成功匹配到的话结束处理逻辑
		if err == nil && isEmpty(result) { //如果采集的内容为空
			err = ErrEmpty
		}
		if err == nil {
			break
		}
	}
	return
}
