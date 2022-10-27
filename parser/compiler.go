package parser

import (
	"errors"

	"git.ziniao.com/webscraper/go-gin-http"
	"git.ziniao.com/webscraper/go-gin-http/parser/parse"
	"git.ziniao.com/webscraper/go-gin-http/proxy"
	"git.ziniao.com/webscraper/go-orm"
	"git.ziniao.com/webscraper/go-orm/log"
)

// 获取模板配置数据资料信息
type CompilerSt struct {
	DocHtml     string
	Device      string
	ParseErr    *parse.ParseError   `json:"-"`
	XPathParser *parse.XPathParseSt `json:"-"`
	QueryParser *parse.QueryParseSt `json:"-"`
	JsonParser  *parse.JsonParseSt  `json:"-"`
}

// 获取生成一个模板编译器
func NewCompiler(doc, device string) *CompilerSt {
	if device != proxy.DEVICE_APP {
		device = proxy.DEVICE_PC
	}
	return &CompilerSt{DocHtml: doc, Device: device}
}

// 获取错误的处理逻辑
func (s *CompilerSt) GetError() *parse.ParseError {
	if s.ParseErr == nil {
		s.ParseErr = parse.NewParseError()
	}
	return s.ParseErr
}

// 模板解析器处理逻辑
func (s *CompilerSt) Parse(link string, elements []parse.ElementSt) (result orm.SqlMap, err error) {
	result = make(orm.SqlMap)
	if elements == nil || len(elements) < 1 {
		log.Write(-1, "解析器模板配置异常...")
		err = errors.New("解析器模块元素获取失败")
		return
	}
	s.ParseErr = parse.NewParseError()
	for _, element := range elements {
		if len(element.Device) > 0 && element.Device != s.Device {
			continue
		}
		errTemp := element.RunParse(s, result)
		if errTemp != nil {
			s.ParseErr.Wrapped(element.Tag, errTemp)
		}
	}
	//如果数据不为空的情况 直接返回空数据信息
	if !s.ParseErr.IsEmpty() { //解析结果有异常的记录日志
		var ckey = ""
		err, ckey = s.ParseErr, "parser@"+core.Md5Str(s.ParseErr.Error())
		core.LogActionOnce(ckey, 86400, err, link, s.DocHtml)
	}
	return
}

// 获取解析数据资料信息
func (s *CompilerSt) ParseErrStats() int {
	if s.ParseErr != nil {
		return s.ParseErr.RequireErr()
	}
	return 0
}

// 解析模块数据资料信息
func (s *CompilerSt) SetDoc(doc string) {
	s.DocHtml = doc
}

// 获取解析匹配的模板
func (s *CompilerSt) GetDoc() string {
	return s.DocHtml
}

// 克隆一个对象返回接口对象
func (s *CompilerSt) Clone(doc string) parse.IFCompiler {
	return parse.IFCompiler(&CompilerSt{DocHtml: doc})
}

// 获取解析器模板引擎
func (s *CompilerSt) GetParser(nodeType int8) parse.IFNodeParser {
	switch nodeType {
	case parse.NODE_TYPE_XPATH:
		return parse.IFNodeParser(s.getXPathParser())
	case parse.NODE_TYPE_QUERY:
		return parse.IFNodeParser(s.getQueryParser())
	case parse.NODE_TYPE_JSON:
		return parse.IFNodeParser(s.getJsonParser())
	case parse.NODE_TYPE_REGREP:
		return parse.IFNodeParser(parse.RegExpParseSt(s.DocHtml))
	}
	panic("get Parser nodeType Not Support")
}

// 获取Json的解析器
func (s *CompilerSt) getJsonParser() *parse.JsonParseSt {
	var err error = nil
	if s.JsonParser == nil {
		s.JsonParser, err = parse.NewJsonParser(s.DocHtml)
		if err != nil { //异常退出 非json的情况
			panic(err)
		}
	}
	return s.JsonParser
}

// 获取go-query的解析器
func (s *CompilerSt) getQueryParser() *parse.QueryParseSt {
	var err error = nil
	if s.QueryParser == nil {
		s.QueryParser, err = parse.NewQueryParse(s.DocHtml)
		if err != nil { //异常退出 非json的情况
			panic(err)
		}
	}
	return s.QueryParser
}

// 获取go-xpath的解析器
func (s *CompilerSt) getXPathParser() *parse.XPathParseSt {
	var err error = nil
	if s.XPathParser == nil {
		s.XPathParser, err = parse.NewXPathParser(s.DocHtml)
		if err != nil { //异常退出 非json的情况
			panic(err)
		}
	}
	return s.XPathParser
}
