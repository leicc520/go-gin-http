package core

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/leicc520/go-orm/log"
)

var gJwtSecret = []byte("xxx-xx.com")

//设置Jtw 的基础用户信息
type JwtUser struct {
	Id int64
	Loginpw string
}

//初始化Jwt秘钥
func init() {
	jwtSecret := os.Getenv("DCJWT")
	if len(jwtSecret) > 0 {
		gJwtSecret = []byte(jwtSecret)
	}
	fmt.Println("====================JWT=========================")
	fmt.Println(string(gJwtSecret))
	fmt.Println("====================JWT=========================")
}

//获取客户端ID 数据信息
func getClientId(clientId string) string {
	if os.Getenv("DCENV") != "prod" {
		clientId = "******"
	} else if len(clientId) > 48 {
		clientId = string([]byte(clientId)[0:48])
	}
	return clientId
}

// 产生token的函数
func JwtToken(id int64, clientId, loginPw string) string {
	clientId = getClientId(clientId)
	idStr   := fmt.Sprintf("%d|%s", id, loginPw)
	expire  := time.Now().Add(30*24*time.Hour).Unix()
	claims  := jwt.StandardClaims{Id: idStr, ExpiresAt: expire, Issuer: "_"}
	tClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	client  := md5.Sum([]byte(clientId))
	secret  := append(gJwtSecret, client[:]...)
	if token, err := tClaims.SignedString(secret); err != nil || len(token) < 1 {
		log.Write(log.ERROR, err)
		return ""
	} else {
		return token
	}
}

// 验证token的函数
func JwtParse(token, clientid string, jwtPtr *JwtUser) error {
	if len(token) < 3 {
		return errors.New("JWT令牌太短了呀！")
	}
	clientid    = getClientId(clientid)
	keyHandle  := func(token *jwt.Token)(interface{},error){
		client := md5.Sum([]byte(clientid))
		secret := append(gJwtSecret, client[:]...)
		return secret, nil
	}
	tClaims, err := jwt.ParseWithClaims(token, &jwt.StandardClaims{}, keyHandle)
	if tClaims != nil && err == nil {
		if sClaims, ok := tClaims.Claims.(*jwt.StandardClaims); ok {
			aStr := strings.SplitN(sClaims.Id, "|", 2)
			if len(aStr) == 2 {//数据不能为空
				jwtPtr.Loginpw = aStr[1] //获取数据信息
				jwtPtr.Id, _ = strconv.ParseInt(aStr[0], 10, 64)
				return nil
			}
		}
		return errors.New("请求的JWT校验不合法.")
	} else {
		log.Write(-1, "请求token异常", token, clientid, string(gJwtSecret), err)
	}
	return err
}
