package micro

import (
	"io/ioutil"
	"os"

	"github.com/leicc520/go-gin-http"
	"github.com/leicc520/go-orm"
	"github.com/leicc520/go-orm/cache"
	"github.com/leicc520/go-orm/log"
	"gopkg.in/yaml.v2"
)

type Config struct {
	App      core.AppConfigSt	   	`yaml:"app"`
	Logger   log.LogFileSt	   		`yaml:"logger"`
	Redis    string 			   	`yaml:"redis"`
	CacheDir string            		`yaml:"cachedir"`
	Cache    cache.CacheConfigSt 	`yaml:"cache"`
}

//加载配置文件数据信息
func (c *Config) Load(confName string, config interface{}) *Config {
	file, err := os.Stat(confName)
	if err == nil && file.Mode().IsRegular() {
		c.LoadFile(confName, config)
	} else {
		c.LoadAddr(confName, config)
	}
	if len(c.CacheDir) > 0 {
		cache.GFileCacheDir = c.CacheDir
	}
	workDir, err := os.Getwd()
	log.Write(-1, "workdir {"+workDir+"} cachedir {"+cache.GFileCacheDir+"}", err)
	return c
}

//加载配置 数据资料信息
func (c *Config)LoadFile(confFile string, config interface{}) *Config {
	if confFile == "" {
		confFile = "config/default.yml"
	}
	if file, err:=os.Stat(confFile); err != nil || !file.Mode().IsRegular() {
		log.Write(log.ERROR, "load Config File Failed: ", err)
	}
	data, _ := ioutil.ReadFile(confFile)
	//把yaml形式的字符串解析成struct类型 先子类初始化
	if err := yaml.Unmarshal(data, config); err != nil {
		log.Write(log.ERROR, "load Config child Parse Failed: ", err)
	}
	//把yaml形式的字符串解析成struct类型 父类加载初始化
	if err := yaml.Unmarshal(data, c); err != nil {
		log.Write(log.ERROR, "load Config parent Parse Failed: ", err)
	}
	return 	c
}

//加载配置 通过配置加载数据
func (c *Config)LoadAddr(srvAddr string, config interface{}) *Config {
	data := core.NewMicroRegSrv(srvAddr).Config(c.App.Name)
	//把yaml形式的字符串解析成struct类型
	if err := yaml.Unmarshal([]byte(data), config); err != nil {
		log.Write(log.ERROR, "load Config child Parse Failed: ", err)
	}
	//把yaml形式的字符串解析成struct类型
	if err := yaml.Unmarshal([]byte(data), c); err != nil {
		log.Write(log.ERROR, "load Config parent Parse Failed", err)
	}
	return 	c
}

//通过配置名称加载配置，然后解析到config配置当中
func (c *Config)LoadConfig(srvAddr, confName string, config interface{}) error {
	data := core.NewMicroRegSrv(srvAddr).Config(confName)
	//把yaml形式的字符串解析成struct类型
	if err := yaml.Unmarshal([]byte(data), config); err != nil {
		return err
	}
	return nil
}

//通过远程配置服务器加载
func (c *Config)LoadDBRemote(dbName string, srvAddr string) {
	data     := core.NewMicroRegSrv(srvAddr).Config(dbName)
	dbSlice  := make([]orm.DbConfig, 0)
	//把yaml形式的字符串解析成struct类型
	if err := yaml.Unmarshal([]byte(data), &dbSlice); err != nil {
		log.Write(log.ERROR, "load Config {"+dbName+"} Parse Failed: ", err)
	}
	for idx := 0; idx < len(dbSlice); idx++ {
		orm.InitDBPoolSt().Set(dbSlice[idx].SKey, &dbSlice[idx])
	}
}

