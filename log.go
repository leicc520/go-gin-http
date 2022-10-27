package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.ziniao.com/webscraper/go-orm"
)

var (
	_baseDirOnceLog = ""
)

// 更新日志数据资料信息
func BaseOnceLogDir(dir string) {
	_baseDirOnceLog = dir
}

// 记录日志，这个动作的内容只记录一次
func LogActionOnce(action string, expire int64, v ...interface{}) {
	cache := orm.GetMCache()
	lStr := "spider@" + Md5Str(action)
	//缓存中已经存在，则不记录
	if isExits := cache.Get(lStr); isExits != nil || len(_baseDirOnceLog) < 1 {
		return
	}
	//相同的内容如果已经记录过了，直接跳过
	cache.Set(lStr, true, expire)
	dir := filepath.Dir(_baseDirOnceLog)
	dir = filepath.Join(dir, "once/"+time.Now().Format(orm.DATEYMDFormat))
	if !orm.FileExists(dir) { //创建目录结构
		os.MkdirAll(dir, 0777)
	}
	file := filepath.Join(dir, action)
	if !strings.Contains(action, "@") {
		file = filepath.Join(dir, lStr)
	}
	file += ".html"
	data := fmt.Sprint(v...) //格式化文件
	os.WriteFile(file, []byte(data), 0777)
}
