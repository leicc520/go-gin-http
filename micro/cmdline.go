package micro

import (
	"flag"
	"os"
	"strings"
	
	jsonIter "github.com/json-iterator/go"
	"github.com/leicc520/go-gin-http"
	"github.com/leicc520/go-orm"
	"github.com/leicc520/go-orm/log"
)

//统计使用json更快速的json解析器
var json = jsonIter.ConfigCompatibleWithStandardLibrary

//初始化命令启动参数
func CmdInit(afterFunc func()) {
	dcEnv, dcJwt, basePath, dcSrv := "", "", "", ""
	flag.StringVar(&dcEnv,"dcEnv", "", "请输入环境变量(prod|smi|dev|loc)...")
	flag.StringVar(&dcSrv,"dcSrv", "", "请输入配置中心(http)地址...")
	flag.StringVar(&dcJwt,"dcJwt", "", "请输入配置中心验证JwtToken...")
	flag.StringVar(&basePath, "basePath", "", "请输入运行家目录路径...")
	flag.Parse() //解析环境变量数据信息
	if len(basePath) > 0 && orm.FileExists(basePath) {
		os.Chdir(basePath)
	} else {
		basePath, _ = os.Getwd()
	}
	if len(dcEnv) > 0 && strings.Contains("prod|smi|dev|loc", dcEnv) {
		os.Setenv(core.DCENV, dcEnv)
	} else {
		dcEnv = os.Getenv(core.DCENV)
	}
	if len(dcJwt) > 0 {
		os.Setenv(core.DCJWT, dcJwt)
	} else {
		dcJwt = os.Getenv(core.DCJWT)
	}
	if len(dcSrv) > 0 {
		os.Setenv(core.DCSRV, dcSrv)
	} else {
		dcSrv = os.Getenv(core.DCSRV)
	}
	if afterFunc != nil { //执行初始化完之后回调处理逻辑
		afterFunc()
	}
	log.Write(-1, "dcEnv", dcEnv, "dcJwt", dcJwt, "dcSrv", dcSrv, "basePath", basePath)
}