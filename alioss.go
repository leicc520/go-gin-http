package core

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"github.com/leicc520/go-orm"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/leicc520/go-orm/log"
)

type AliOssSt struct {
	AccessKeyId     string `yaml:"accesskeyid"`
	AccessKeySecret string `yaml:"accesskeysecret"`
	Endpoint        string `yaml:"endpoint"`
	OssHost         string `yaml:"osshost"`
	Bucket          string `yaml:"bucket"`
	BaseUrl         string `yaml:"baseurl"`
	IsPush          string `yaml:"ispush"`
}

// 获取上传服务签名数据信息
func (s AliOssSt) UploadSign(dir string, sizeMB, expire int64) interface{} {
	expire += time.Now().Unix()
	sizeMB = sizeMB * 1024 * 1024
	condSize := []interface{}{"content-length-range", 0, sizeMB}
	condPrefix := []interface{}{"starts-with", "$key", dir}
	params := map[string]interface{}{"expiration": orm.TimeStampFormat(expire, "2006-01-02T15:04:05Z"),
		"conditions": []interface{}{condSize, condPrefix}}
	paramsByte, _ := json.Marshal(params)
	policy := base64.StdEncoding.EncodeToString(paramsByte)
	mac := hmac.New(sha1.New, []byte(s.AccessKeySecret))
	mac.Write([]byte(policy))
	signStr := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	dataSet := map[string]interface{}{"accessid": s.AccessKeyId, "policy": policy,
		"host": s.OssHost, "expire": expire, "dir": dir, "size": sizeMB, "signature": signStr}
	return dataSet
}

// oss上传文件的处理逻辑
func AliOssUpfile(file, ossPath string, config *AliOssSt) error {
	if config.IsPush != "on" {
		return nil //系统不关闭的情况
	}
	client, err := oss.New(config.Endpoint,
		config.AccessKeyId, config.AccessKeySecret)
	if err != nil {
		log.Write(log.ERROR, "create alioss error", err)
		return err
	}
	// 获取存储空间。
	bucket, err := client.Bucket(config.Bucket)
	if err != nil {
		log.Write(log.ERROR, "get bucket error", err)
		return err
	}
	//上传本地文件。
	ossPath = strings.TrimLeft(ossPath, "/")
	if err = bucket.PutObjectFromFile(ossPath, file); err != nil {
		log.Write(log.ERROR, "alioss upload file error", err)
		return err
	}
	return nil
}
