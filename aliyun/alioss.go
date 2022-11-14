package aliyun

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"strings"
	"time"

	"git.ziniao.com/webscraper/go-orm"
	"git.ziniao.com/webscraper/go-orm/log"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type AliOssSt struct {
	AccessKeyId     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	Endpoint        string `yaml:"endpoint"`
	OssHost         string `yaml:"oss_host"`
	Bucket          string `yaml:"bucket"`
	BaseUrl         string `yaml:"baseurl"`
	IsPush          string `yaml:"is_push"`
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

// 获取绝对地址处理逻辑
func (s AliOssSt) Link(ossPath string) string {
	if strings.HasPrefix(ossPath, "http") {
		return ossPath
	}
	if strings.HasPrefix(s.BaseUrl, "http") {
		return s.BaseUrl + ossPath
	}
	if strings.HasPrefix(s.OssHost, "http") {
		return s.BaseUrl + ossPath
	}
	return s.Endpoint + ossPath
}

// 获取ACL的请求地址 私密bucket获取地址
func (s AliOssSt) GetFile(ossPath string) (string, error) {
	_, bucket, err := s.init("on")
	if err != nil { //系统不关闭的情况
		return "", err
	}
	//上传本地文件。
	ossPath = strings.TrimLeft(ossPath, "/")
	return bucket.SignURL(ossPath, oss.HTTPGet, 86400, nil)
}

// oss上传文件的处理逻辑
func (s AliOssSt) AliOssUpfile(file, ossPath string) error {
	_, bucket, err := s.init(s.IsPush)
	if err != nil || bucket == nil { //系统不关闭的情况
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

// 初始化bucket数据信息
func (s AliOssSt) init(isPush string) (*oss.Client, *oss.Bucket, error) {
	if isPush != "on" {
		return nil, nil, nil //系统不关闭的情况
	}
	client, err := oss.New(s.Endpoint, s.AccessKeyId, s.AccessKeySecret)
	if err != nil {
		log.Write(log.ERROR, "create alioss error", err)
		return nil, nil, err
	}
	// 获取存储空间。
	bucket, err := client.Bucket(s.Bucket)
	if err != nil {
		log.Write(log.ERROR, "get bucket error", err)
		return nil, nil, err
	}
	return client, bucket, nil
}

// 上传文件的出来逻辑，直接通过目标上传
func (s AliOssSt) AliOssUpfileV2(file *multipart.FileHeader, ossPath string) error {
	fd, err := file.Open()
	if err != nil { //打开文件失败的情况
		return err
	}
	defer fd.Close()
	_, bucket, err := s.init(s.IsPush)
	if err != nil || bucket == nil { //系统不关闭的情况
		return err
	}
	//上传本地文件。
	ossPath = strings.TrimLeft(ossPath, "/")
	if err = bucket.PutObject(ossPath, fd); err != nil {
		log.Write(log.ERROR, "alioss upload file error", err)
		return err
	}
	return nil
}
