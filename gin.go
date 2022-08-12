package core

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
	
	"github.com/gin-gonic/gin"
	jsonIter "github.com/json-iterator/go"
	"github.com/leicc520/go-gin-http/tracing"
	"github.com/leicc520/go-orm"
	"github.com/leicc520/go-orm/log"
)

const (
	JwtHeader   = "SIGNATURE"
	JwtQuery    = "_s"
	JwtCookie   = "_s"
	EncryptName = "cryptSt"
	EncryptKeys = "X-KEY"
)

type AppConfigSt struct {
	Host 		string 	`yaml:"host"`
	Name 		string 	`yaml:"name"`
	Ssl 		string 	`yaml:"ssl"`
	Version 	string 	`yaml:"version"`
	ImSeg 		string 	`yaml:"im"`
	Domain 		string 	`yaml:"domain"` //网站的域名
	CertFile 	string 	`yaml:"certFile"`
	KeyFile 	string 	`yaml:"keyFile"`
	CrossDomain string 	`yaml:"crossDomain"`
	Tracing 	tracing.JaegerTracingConfigSt `yaml:"tracing"`
	UpFileDir 	string 	`yaml:"upfileDir"`
	UpFileBase  string 	`yaml:"upfileBase"`
}

type AppStartHandler func(c *gin.Engine)
type Application struct {
	app    *gin.Engine
	baseUrl string
	srvHost string
	config *AppConfigSt
	handler []AppStartHandler
}

var (
	app *Application = nil
	json = jsonIter.ConfigCompatibleWithStandardLibrary
)

//初始化创建一个http服务的情况
func NewApp(config *AppConfigSt) *Application {
	app = &Application{app: gin.New(), handler: make([]AppStartHandler, 0), config: config}
	app.app.Use(gin.Logger(), GINRecovery())
	if config.Tracing.IsTracing() && nil != (&config.Tracing).Init(config.Name) {
		app.app.Use(GINTracing()) //有配置的话开启链路跟踪
	}
	if strings.ToLower(config.CrossDomain) == "on" {
		app.app.Use(GINCors()) //跨域的支持集成
	}
	app.app.GET("/tracing", handleTracing)
	app.app.GET("/healthz", func(c *gin.Context) {
		c.String(200, config.Version)
	})
	GinValidatorInit("zh")
	return app
}

//初始化协议http协议
func (app *Application) httpProto() (string, string, bool) {
	httpStr, wsStr, isSsl := "", "", false
	isSsl = strings.HasPrefix(strings.ToLower(app.config.Ssl), "on")
	if isSsl && orm.FileExists(app.config.KeyFile) && orm.FileExists(app.config.KeyFile) {
		httpStr = "https://"+app.config.Host
		if len(app.config.ImSeg) > 1 {
			wsStr = "wss://"+app.config.Host+app.config.ImSeg
		}
	} else {//配置阐述不对的情况
		isSsl   = false
		httpStr = "http://"+app.config.Host
		if len(app.config.ImSeg) > 1 {
			wsStr = "ws://"+app.config.Host+app.config.ImSeg
		}
	}
	//如果设置的开启微服务注册的情况，需要主动注册一下微服务
	if RegSrv != nil && len(RegSrv.GetRegSrv()) > 1 {
		time.AfterFunc(time.Second*3, func() {
			app.srvHost = RegSrv.Register(app.config.Name, app.config.Host,"http", app.config.Version)
		})
	}
	app.baseUrl = httpStr
	return httpStr, wsStr, isSsl
}

//释放资源处理逻辑业务 服务注销等等
func (app *Application) release()  {
	fmt.Println("==============释放资源退出=================")
	orm.GdbPoolSt.Release()
	if RegSrv != nil && len(RegSrv.GetRegSrv()) > 1 {
		RegSrv.UnRegister("http", app.config.Name, app.srvHost)
	}
}

//注册预先要执行的业务动作处理逻辑
func (app *Application) RegHandler(handler AppStartHandler) *Application {
	app.handler = append(app.handler, handler)
	return app
}

/***************************************************************************
 服务的管理放到的linux/windows当中，因为不同系统对优雅启动的支出不一致
 */
func ShouldBind(c *gin.Context, obj interface{}) error {
	sKey := c.GetHeader(EncryptKeys)
	if len(sKey) > 6 {//解密的数据业务处理逻辑
		cryptSt := Crypt{JKey: []byte(sKey)}
		c.Set(EncryptName, cryptSt) //设置数据解码
		oldStr, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Write(log.ERROR, "读取请求数据异常", err)
			return err
		}
		log.Write(log.INFO, "数据接收:", string(oldStr))
		defer c.Request.Body.Close()
		req := cryptSt.Decrypt(oldStr)
		log.Write(log.INFO, "数据解码:", string(req))
		if req == nil || len(req) < 1 {
			return errors.New("数据解码失败,无法操作.")
		}
		if err = json.Unmarshal(req, obj); err != nil {
			log.Write(log.ERROR, "数据解码:", string(req), err)
			return err
		}
		if err = ValidateStruct(obj); err != nil {
			log.Write(log.ERROR, "结构校验:", string(req), err)
			return err
		}
	} else {//非加密的业务处理逻辑
		if err := c.ShouldBind(&obj); err != nil {
			return  err
		}
	}
	return nil
}

//获取请求的token数据资料信息
func JWTACLToken(c *gin.Context) string {
	token := c.GetHeader(JwtHeader)
	if len(token) < 3 {
		token, _ = c.Cookie(JwtCookie)
		if len(token) < 3 {
			token = c.Query(JwtQuery)
		}
	}
	return token
}

//获取JWT登录授权校验uid
func JWTACLUserid(c *gin.Context) int64 {
	token, signUser := JWTACLToken(c), JwtUser{} //设置初始化信息
	if err := JwtParse(token, c.Request.UserAgent(), &signUser); err != nil {
		return -1
	}
	return signUser.Id
}

//计算接口请求的执行时间 并做业务错误拦截处理
func GINRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		sTime  := time.Now()
		defer func() {
			log.Write(log.DEBUG, c.Request.RequestURI, "执行时间:", time.Since(sTime))
			if err := recover(); err != nil {//执行panic数据恢复处理逻辑
				view := NewHttpView(c)
				if o, ok := err.(HttpError); ok {
					view.ErrorDisplay(o.Code, o.Msg)
				} else {//未知的错误情况处理逻辑
					rtStack := orm.RuntimeStack(3)
					errStr, _  := json.Marshal(err)
					log.Write(log.ERROR, c.Request.RequestURI,  c.Request.UserAgent(), JWTACLToken(c))
					log.Write(log.ERROR, "GINRecovery", string(errStr), string(rtStack))
					view.ErrorDisplay(500, "内部服务错误,拒绝服务")
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}

//HTTP请求跨域的业务处理逻辑
func GINCors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")                                       // 这是允许访问所有域
		c.Header("Access-Control-Allow-Methods", "POST,GET,OPTIONS,PUT,DELETE,UPDATE") //服务器支持的所有跨域请求的方法,为了避免浏览次请求的多次'预检'请求
		c.Header("Access-Control-Allow-Headers", "SIGNATURE, Content-Length, X_Requested_With, X-KEY, Accept, Origin, Host, Accept-Encoding, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma")
		c.Header("P3P", "CP=\"CURa ADMa DEVa PSAo PSDo OUR BUS UNI PUR INT DEM STA PRE COM NAV OTC NOI DSP COR\"")
		c.Header("Access-Control-Allow-Credentials", "false")
		if strings.ToUpper(c.Request.Method) == "OPTIONS" {
			c.AbortWithStatus(202)
			return
		}
		c.Next() //  处理请求
	}
}

//验证码登录 请求token 数据信息
func GINJWTCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, signUser := JWTACLToken(c), JwtUser{} //设置初始化信息
		if err := JwtParse(token, c.Request.UserAgent(), &signUser); err != nil {
			NewHttpView(c).ErrorDisplay(9999, "请求token异常")
			c.Abort()
			return
		}
		c.Set("user", &signUser) //设置请求的用户ID
		c.Next()
	}
}