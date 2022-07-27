package core

import (
	"crypto/md5"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/leicc520/go-orm/log"
)

//获取相对路径的截取
func RelativePath(absPath string) string {
	var baseDir = ""
	if coConfig != nil { //路径不存在的情况
		path, err := filepath.Abs(coConfig.UpFileBase)
		if err != nil {
			log.Write(log.ERROR, "基础目录获取绝对路径出错 "+err.Error())
			return absPath
		}
		baseDir = path
	} else { //默认取项目的执行目录
		baseDir, _ = os.Getwd()
	}
	if len(baseDir) > len(absPath) {
		log.Write(log.ERROR, "基础路径长度大于绝对路径长度")
		return absPath
	}
	absPath = strings.ReplaceAll(absPath[len(baseDir):], string(os.PathSeparator), "/")
	return absPath
}

//获取上传目录的文件
func FilePathBuild(appdir string, FILE *multipart.FileHeader, subfix string) (string, error) {
	var fileName string = ""
	if fp, err := FILE.Open(); err == nil { //读取内容做hash
		defer fp.Close()
		readByte := make([]byte, 4096)
		if n, err := fp.Read(readByte); err == nil && n > 1024 {
			fileName = fmt.Sprintf("%x", md5.Sum(readByte))
		}
	}
	if len(fileName) < 16 { //文件名截取失败的情况
		fileName = fmt.Sprintf("%x", md5.Sum([]byte(FILE.Filename)))
		fileName = time.Now().Format("20060102150405") + fileName[:8]
	}
	pathFile, err := filepath.Abs(coConfig.UpFileDir) //上传目录基础路径
	if err != nil {                                   //数据获取为空的情况
		log.Write(log.ERROR, coConfig.UpFileDir+" 获取绝对路径失败"+err.Error())
		return "", err
	}
	if os.Getenv("DCENV") != "prod" { //非生产环境的情况
		pathFile = filepath.Join(pathFile, "test")
	}
	if ok, err := regexp.MatchString(`^[a-z]+$`, appdir); ok && err == nil {
		pathFile = filepath.Join(pathFile, appdir)
	}
	pathFile = filepath.Join(pathFile, time.Now().Format("200601"))
	if fp, err := os.Stat(pathFile); err != nil || !fp.IsDir() {
		os.MkdirAll(pathFile, 0777)
	}
	npos := strings.LastIndex(FILE.Filename, ".")
	if npos > 0 { //拼接扩展名
		fileName += FILE.Filename[npos:]
	} else { //使用默认的后缀
		fileName += subfix
	}
	pathFile = filepath.Join(pathFile, fileName)
	return pathFile, nil
}
