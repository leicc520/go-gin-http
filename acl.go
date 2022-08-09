package core

import (
	"crypto/md5"
	"fmt"
	"time"
	
	"github.com/leicc520/go-orm"
)

const coCryptKey = "%^&*f1l5ds%3#cx"

type sysSafeSt struct {
	Tocken  string
	Loginpw string
	Expire  int64
}

type AclSt struct {
	Sys     int8
	Ckey    string
	DBSafe *orm.ModelSt
}

/********************************************************************
CREATE TABLE `sys_safe` (
  `userid` int(10) unsigned NOT NULL COMMENT '账号ID',
  `sys` tinyint(1) NOT NULL COMMENT '系统别0-web 1-app',
  `loginpw` varchar(63) DEFAULT NULL COMMENT '会员密码生成的Tocken',
  `tocken` varchar(32) DEFAULT NULL COMMENT '随机码生成的Tocken',
  `expire` int(10) unsigned DEFAULT NULL COMMENT '过期时间',
  PRIMARY KEY (`userid`,`sys`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC COMMENT='安全Tocken'
 */
func NewAcl(sys int8, dbSafeSt *orm.ModelSt) *AclSt {
	return &AclSt{Sys: sys, DBSafe: dbSafeSt}
}

func (a *AclSt) Skey(id int64) string {
	return fmt.Sprintf("acl@%d-%d", id, a.Sys)
}

//登录密码的加密
func (a *AclSt) Crypt(loginPw string) string {
	aStr := coCryptKey + loginPw + coCryptKey
	aStr = fmt.Sprintf("%x", md5.Sum([]byte(aStr)))
	return aStr
}

//设置请求密钥token信息 --设置请求的token信息
func (a *AclSt) SetToken(id int64, loginPw, xToken string, expire int64) string {
	sTime := time.Now().Unix()
	if expire > 0 && expire < sTime {
		expire += sTime
	}
	oToken := loginPw + "-" + xToken
	if a.DBSafe != nil {//设置了安全登录的情况
		a.DBSafe.SetModTable(id).NewOneFromHandler(func(st *orm.QuerySt) *orm.QuerySt {
			st.Value("tocken", xToken).Value("loginpw", loginPw)
			st.Value("userid", id).Value("sys", a.Sys).Value("expire", expire)
			return st
		}, func(st *orm.QuerySt) interface{} {
			st.Duplicate("tocken", xToken).Duplicate("loginpw", loginPw).Duplicate("expire", expire)
			return nil
		})
	}
	oToken = fmt.Sprintf("%x", md5.Sum([]byte(oToken)))
	if orm.GdbCache != nil { //设置缓存处理逻辑
		orm.GdbCache.Set(a.Skey(id), oToken, expire)
	}
	return oToken
}

//获取请求加密的token
func (a *AclSt) GetToken(id int64) string {
	if orm.GdbCache != nil {
		datas := orm.GdbCache.Get(a.Skey(id))
		if datas != nil {
			return fmt.Sprintf("%v", datas)
		}
	}
	if a.DBSafe != nil { //不为空的情况
		xAcl  := sysSafeSt{}
		err   := a.DBSafe.SetModTable(id).GetItem(func(st *orm.QuerySt) string {
			st.Where("sys", a.Sys).Where("userid", id)
			return st.GetWheres()
		}, "tocken,loginpw,expire").ToStruct(&xAcl)
		if err == nil && (xAcl.Expire < 1 || xAcl.Expire > time.Now().Unix())  { //数据为空的情况
			oToken := fmt.Sprintf("%x", md5.Sum([]byte(xAcl.Loginpw+"-"+xAcl.Tocken)))
			if orm.GdbCache != nil { //设置缓存处理逻辑
				orm.GdbCache.Set(a.Skey(id), oToken, xAcl.Expire)
			}
			return oToken
		}
	}
	return ""
}