package core

import (
	"strings"
)

var errCode = map[int]string{
	0:    "OK",
	500:  "内部服务错误,拒绝服务",
	1001: "参数校验,拒绝访问",
	1002: "文件上传发生未知异常",
	1003: "上传文件路径生成失败",
	1004: "文件保存处理失败",
	1005: "请求token验证失败",
	1006: "请求参数异常,无法操作",
	1007: "用户数据异常,稍后再试",
	1008: "权限不足,无法完成操作",
	1009: "SQL执行出错,请检查SQL",
	1010: "验证码校验错误",
	1011: "账号被禁用,无法登录",
	1012: "账号密码错误,无法登录",
	1013: "账号已经过期,无法登录",
	1014: "记录有关联信息,不允许删除",
	1015: "编码已经存在,不允许重复添加",
	1016: "记录已存在,请勿重复添加",
}

//创建一个错误
func NewHttpError(code int, msg ...string) HttpError {
	mStr := strings.Join(msg, ",")
	if len(mStr) < 3 {
		ok := false
		if mStr, ok = errCode[code]; !ok {
			mStr = "未知错误,请重试"
		}
	}
	return HttpError{Code: code, Msg: mStr}
}

//标准化输出 -直接输出错误 然后recover当中做拦截
func PanicValidateHttpError(code int, err error) {
	argsErr := NewHttpError(code)
	if err != nil {
		argsErr.SetDebug(err.Error()) //具体的报错提示
	}
	if transErr := ValidateTranslator(err, nil); transErr != nil {
		argsErr.Msg = transErr.Error() //具体的报错提示
	}
	panic(argsErr)
}

//标准化输出 -直接输出错误 然后recover当中做拦截
func PanicHttpError(code int, msg ...string) {
	panic(NewHttpError(code, msg...))
}