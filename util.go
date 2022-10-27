package core

import (
	"crypto/md5"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/leicc520/go-orm/log"
)

// 读取文件内容数据信息
func ReadFile(file string) string {
	stream, err := os.ReadFile(file)
	if err != nil {
		log.Write(log.ERROR, "file read error", err)
	}
	return string(stream)
}

// 清理html 样式表和js代码
func HTMLClean(htmlStr string) string {
	htmlStr = regexp.MustCompile(`(?s)<style.*?>.*?</style>`).ReplaceAllString(htmlStr, "")
	htmlStr = regexp.MustCompile(`(?s)<noscript.*?>.*?</noscript>`).ReplaceAllString(htmlStr, "")
	htmlStr = regexp.MustCompile(`(?s)<script.*?>.*?</script>`).ReplaceAllString(htmlStr, "")
	return htmlStr
}

// 过滤字符串处理逻辑
func StripQuotes(str string) string {
	if (strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}")) || (strings.HasPrefix(str, "[") && strings.HasSuffix(str, "]")) {
		str = strings.ReplaceAll(str, "{&#34;", "{\"")
		str = strings.ReplaceAll(str, "&#34;}", "\"}")
		str = strings.ReplaceAll(str, "&#34;:", "\":")
		str = strings.ReplaceAll(str, ":&#34;", ":\"")
		str = strings.ReplaceAll(str, "&#34;,", "\",")
		str = strings.ReplaceAll(str, ",&#34;", ",\"")
		str = strings.ReplaceAll(str, "[&#34;", "[\"")
		str = strings.ReplaceAll(str, "&#34;]", "\"]")
	}
	return str
}

// 过滤html标签处理逻辑
func StripTags(htmlStr string) string {
	htmlStr = strings.ReplaceAll(htmlStr, "&gt;", ">")
	htmlStr = strings.ReplaceAll(htmlStr, "&lt;", "<")
	htmlStr = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(htmlStr, "")
	htmlStr = strings.ReplaceAll(htmlStr, "&amp;", "&")
	htmlStr = strings.ReplaceAll(htmlStr, "&#39;", "'")
	htmlStr = strings.TrimSpace(htmlStr)
	//针对json格式数据的过滤处理逻辑
	htmlStr = StripQuotes(htmlStr)
	//去掉特色字符 --直接过滤掉 不能太宽松,不然易出错
	htmlStr = regexp.MustCompile(`&[#]?[0-9a-zA-Z]{1,5};`).ReplaceAllString(htmlStr, "")
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(htmlStr, " "))
}

// 字符串截取字段逻辑
func CutStr(str string, length int, suffix string) string {
	s := []rune(str)
	total := len(s)
	if total <= length {
		return str
	}
	if length < 0 {
		length = total
	}
	result := string(s[0:length]) + suffix
	return result
}

// 获取字符串md5 hash值
func Md5Str(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

// 格式化json数据资料信息
func PrettyJson(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b)
}

// 切片转字符串
func Slice2Str(l interface{}, sep string) string {
	if l == nil {
		return ""
	}
	//普通的字符串 直接返回即可
	if aStr, ok := l.([]string); ok {
		return strings.Join(aStr, sep)
	}
	v := reflect.ValueOf(l)
	if !v.IsValid() || v.Kind() != reflect.Slice {
		return ""
	}
	result := make([]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = fmt.Sprintf("%v", v.Index(i))
	}
	return strings.Join(result, sep)
}

// 获取文件的后缀名称
func FileSuffix(file string) string {
	npos := strings.LastIndex(file, ".")
	if npos > 0 {
		return file[npos:]
	}
	return ""
}

// 切片拆分
func ArraySplitGroupsOf(arr []string, num int64) [][]string {
	max := int64(len(arr))
	//判断数组大小是否小于等于指定分割大小的值，是则把原数组放入二维数组返回
	if max <= num {
		return [][]string{arr}
	}
	//获取应该数组分割为多少份
	var quantity int64
	if max%num == 0 {
		quantity = max / num
	} else {
		quantity = (max / num) + 1
	}
	//声明分割好的二维数组
	var segments = make([][]string, 0)
	//声明分割数组的截止下标
	var start, end, i int64
	for i = 1; i <= quantity; i++ {
		end = i * num
		if i != quantity {
			segments = append(segments, arr[start:end])
		} else {
			segments = append(segments, arr[start:])
		}
		start = i * num
	}
	return segments
}
