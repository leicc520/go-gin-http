package parse

import (
	"git.ziniao.com/webscraper/go-orm/log"
	"strings"
)

type ParseError struct {
	message     []string
	requiredErr int
	totalErr    int
}

// 构造函数
func NewParseError() *ParseError {
	return &ParseError{
		message: make([]string, 0),
	}
}

// 获取错误数据信息收集
func (e *ParseError) Error() string {
	return strings.Join(e.message, "\r\n")
}

// 判定是否为空的情况
func (e *ParseError) IsEmpty() bool {
	if len(e.message) > 0 {
		log.Write(log.DEBUG, e.requiredErr, e.message)
	}
	return e.requiredErr < 1
}

// 返回总的错误次数
func (e *ParseError) TotalErr() int {
	return e.totalErr
}

// 返回总的错误次数
func (e *ParseError) RequireErr() int {
	return e.requiredErr
}

// 包裹错误数据信息
func (e *ParseError) Wrapped(field string, err error) {
	e.totalErr += 1
	e.message = append(e.message, field+"="+err.Error())
	if err != ErrOmitting {
		e.requiredErr++
	}
}
