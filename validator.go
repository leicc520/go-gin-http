package core

import (
	"errors"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	"github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	chTranslations "github.com/go-playground/validator/v10/translations/zh"
)

var gLocale string = "zh"
var  gTranslater ut.Translator = nil
//初始化验证器翻译
func GinValidatorInit(locale string) {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		//注册一个获取json tag的自定义方法
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
		if gTranslater == nil {
			en, cn := en.New(), zh.New()
			uni := ut.New(en, cn, en) //第一个备用
			gTranslater, _ = uni.GetTranslator(locale)
		}
		// 添加额外翻译
		_ = v.RegisterTranslation("required_with", gTranslater, func(ut ut.Translator) error {
			return ut.Add("required_with", "{0} 为必填字段!", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("required_with", fe.Field())
			return t
		})
		_ = v.RegisterTranslation("required_without", gTranslater, func(ut ut.Translator) error {
			return ut.Add("required_without", "{0} 为必填字段!", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("required_without", fe.Field())
			return t
		})
		_ = v.RegisterTranslation("required_without_all", gTranslater, func(ut ut.Translator) error {
			return ut.Add("required_without_all", "{0} 为必填字段!", true)
		}, func(ut ut.Translator, fe validator.FieldError) string {
			t, _ := ut.T("required_without_all", fe.Field())
			return t
		})
		// 注册翻译器
		if locale == "zh" {
			chTranslations.RegisterDefaultTranslations(v, gTranslater)
		} else {
			enTranslations.RegisterDefaultTranslations(v, gTranslater)
		}
	}
}

func ValidateSetLocale(locale string) {
	gLocale = locale
}

//查询信息翻译
func ValidateTranslator(err error, trans ut.Translator) error {
	if trans == nil {//全局数据信息
		trans = gTranslater
	}
	if vErrors, ok := err.(validator.ValidationErrors); ok {
		for _, err0 := range vErrors {
			err = errors.New(err0.Translate(trans))
			return err
		}
	}
	return nil
}

//请求结构提参数验证出来逻辑
func ValidateStruct(st interface{}) error {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		return v.Struct(st)
	}
	return nil
}


