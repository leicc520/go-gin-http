package micro

import (
	"flag"
	"os"
	"strings"

	"github.com/leicc520/go-orm"
	"github.com/leicc520/go-orm/log"
)

const (
	DCSRV = "DCSRV"
	DCJWT = "DCJWT"
	DCENV = "DCENV"
)

//初始化命令启动参数
func CmdInit() {
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
		os.Setenv(DCENV, dcEnv)
	} else {
		dcEnv = os.Getenv(DCENV)
	}
	if len(dcJwt) > 0 {
		os.Setenv(DCJWT, dcJwt)
	} else {
		dcJwt = os.Getenv(DCJWT)
	}
	if len(dcSrv) > 0 {
		os.Setenv(DCSRV, dcSrv)
	} else {
		dcSrv = os.Getenv(DCSRV)
	}
	log.Write(-1, "dcEnv", dcEnv, "dcJwt", dcJwt, "dcSrv", dcSrv, "basePath", basePath)
}