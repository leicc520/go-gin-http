package core

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/leicc520/go-orm"
	"github.com/leicc520/go-orm/cache"
	"github.com/leicc520/go-orm/log"
	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
)

/****************************************************************************************
这里默认使用内存存储验证码的信息，分布式部署的时候可以切换到redis，否则可能验证有问题.
*/
type MemStore struct {
	Store cache.Cacher
}

type CaptchaSt struct {
}

const CapCookie = "_cap"
var Gcaptcha *CaptchaSt = nil

//延迟执行获取验证码存储的初始化
func NewInitCap()  {
	Gcaptcha = &CaptchaSt{}
	time.AfterFunc(time.Second*3, func() {
		store := &MemStore{Store: orm.GdbCache}
		captcha.SetCustomStore(store)
		log.Write(-1, "初始化验证码...")
	})
}

//设置验证的缓存默认1小时过期
func (c *MemStore) Set(id string, digits []byte) {
	c.Store.Set("captcha@"+id, string(digits), 3600)
}

func (c *MemStore) Get(id string, clear bool) []byte {
	data := c.Store.Get("captcha@" + id)
	if clear { //删除记录
		c.Store.Del("captcha@" + id)
	}
	if lstr, ok := data.(string); ok {
		return []byte(lstr)
	}
	return nil
}

//生成验证码的处理逻辑
func (s *CaptchaSt) Serve(w http.ResponseWriter, r *http.Request, id, ext, lang string, download bool, width, height int) error {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	var content bytes.Buffer
	switch ext {
	case ".png":
		w.Header().Set("Content-Type", "image/png")
		err := captcha.WriteImage(&content, id, width, height)
		fmt.Println(err)
	case ".wav":
		w.Header().Set("Content-Type", "audio/x-wav")
		captcha.WriteAudio(&content, id, lang)
	default:
		return captcha.ErrNotFound
	}
	if download {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	http.ServeContent(w, r, id+ext, time.Time{}, bytes.NewReader(content.Bytes()))
	return nil
}

func (s *CaptchaSt)CheckCaptchaSum(sumId string) string {
	aStr := strings.SplitN(sumId, "-", 2)
	if aStr != nil && len(aStr) == 2 {
		hStr := fmt.Sprintf("%x", md5.Sum([]byte(aStr[1])))
		if aStr[0] == hStr[0:6] {
			return aStr[1]
		}
	}
	return ""
}

func (s *CaptchaSt)CaptchaSum(id string) string {
	hStr := fmt.Sprintf("%x", md5.Sum([]byte(id)))
	return hStr[0:6] + "-" + id
}

//执行验证码的处理逻辑
func (s *CaptchaSt)CheckSum(sumid, vcode string) bool {
	idStr := s.CheckCaptchaSum(sumid)
	if idStr == "" || !captcha.VerifyString(idStr, vcode) {
		return false
	}
	return true
}

//生成请求的hash数值
func (s *CaptchaSt)GenerateHash(c *gin.Context) string {
	sTime := time.Now().Unix()
	aStr := c.Request.UserAgent()
	aStr = fmt.Sprintf("%d,%s", sTime, aStr)
	xStr := fmt.Sprintf("%x", md5.Sum([]byte(aStr)))
	aStr = fmt.Sprintf("%s-%d-%s", xStr[0:5], sTime, xStr[27:])
	return aStr
}

//验证请求的hash是否合法
func (s *CaptchaSt)CheckHash(c *gin.Context, xStr string) bool {
	aStr := c.Request.UserAgent()
	vStr := strings.Split(xStr, "-")
	if vStr == nil || len(vStr) != 3 {
		return false
	}
	aStr = fmt.Sprintf("%s,%s", vStr[1], aStr)
	xStr = fmt.Sprintf("%x", md5.Sum([]byte(aStr)))
	if vStr[0] == xStr[0:5] && vStr[2] == xStr[27:] {
		sTime, err := strconv.ParseInt(vStr[1], 10, 64)
		if err == nil && time.Now().Unix()-sTime < 10 {
			return true
		}
	}
	return false
}