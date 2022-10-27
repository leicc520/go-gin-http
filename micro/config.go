package micro

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"git.ziniao.com/webscraper/go-gin-http"
	"git.ziniao.com/webscraper/go-orm"
	"git.ziniao.com/webscraper/go-orm/cache"
	"git.ziniao.com/webscraper/go-orm/log"
)

type Config struct {
	App      core.AppConfigSt    `yaml:"app"`
	Logger   log.LogFileSt       `yaml:"logger"`
	Redis    string              `yaml:"redis"`
	CacheDir string              `yaml:"cachedir"`
	Cache    cache.CacheConfigSt `yaml:"cache"`
}

// 加载配置文件数据信息
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
	log.SetLogger(c.Logger.Init())
	orm.InitDBPoolSt().LoadDbConfig(config) //配置数据库结构注册到数据库调用配置当中
	log.Write(-1, "workdir {"+workDir+"} cachedir {"+cache.GFileCacheDir+"}", err)
	return c
}

// 加载配置 数据资料信息
func LoadFile(confFile string, config interface{}) ([]byte, error) {
	if confFile == "" {
		confFile = "config/default.yml"
	}
	if file, err := os.Stat(confFile); err != nil || !file.Mode().IsRegular() {
		return nil, err
	}
	data, _ := os.ReadFile(confFile)
	data = []byte(envYamlReplace(string(data))) //检测环境变量替换
	//把yaml形式的字符串解析成struct类型 先子类初始化
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}
	return data, nil
}

// 检测yaml文件内容，替换环境变量
func envYamlReplace(str string) string {
	regEnv, _ := regexp.Compile("\\${[\\s]*([^}]+)[\\s]*}")
	arrItems := regEnv.FindAllStringSubmatch(str, -1)
	if arrItems == nil || len(arrItems) < 1 {
		return str
	}
	for _, aStr := range arrItems {
		if len(aStr) != 2 {
			continue
		}
		envStr := os.Getenv(strings.TrimSpace(aStr[1])) //获取环境变量Key信息
		str = strings.Replace(str, aStr[0], envStr, -1)
	}
	writeCacheConfig(str)
	return str
}

// 写入文件缓存的策略
func writeCacheConfig(str string) {
	dir, _ := os.Getwd()
	file := filepath.Join(dir, "cachedir", "cache", "config")
	os.MkdirAll(file, 0777)
	file = filepath.Join(file, orm.RandString(8))
	os.WriteFile(file, []byte(str), 0777)
}

// 加载配置 数据资料信息
func (c *Config) LoadFile(confFile string, config interface{}) *Config {
	if data, err := LoadFile(confFile, config); err != nil {
		log.Write(log.ERROR, "load Config File Failed: ", err)
	} else { //把yaml形式的字符串解析成struct类型 父类加载初始化
		if err = yaml.Unmarshal(data, c); err != nil {
			log.Write(log.ERROR, "load Config parent Parse Failed: ", err)
		}
	}
	return c
}

// 加载配置 通过配置加载数据
func (c *Config) LoadAddr(srvAddr string, config interface{}) *Config {
	if data, err := LoadAddr(srvAddr, c.App.Name, config); err != nil {
		log.Write(log.ERROR, "load Config child Parse Failed: ", err)
	} else { //把yaml形式的字符串解析成struct类型
		if err = yaml.Unmarshal([]byte(data), c); err != nil {
			log.Write(log.ERROR, "load Config parent Parse Failed", err)
		}
	}
	return c
}

// 通过配置名称加载配置，然后解析到config配置当中
func LoadAddr(srvAddr, appName string, config interface{}) (string, error) {
	data := NewRegSrvClient(srvAddr).Config(appName)
	data = envYamlReplace(data) //检测环境变量替换
	//把yaml形式的字符串解析成struct类型
	if err := yaml.Unmarshal([]byte(data), config); err != nil {
		return "", err
	}
	return data, nil
}

// 通过远程配置服务器加载
func (c *Config) LoadDBRemote(dbName string, srvAddr string) {
	data := NewRegSrvClient(srvAddr).Config(dbName)
	dbSlice := make([]orm.DbConfig, 0)
	//把yaml形式的字符串解析成struct类型
	data = envYamlReplace(data) //检测环境变量替换
	if err := yaml.Unmarshal([]byte(data), &dbSlice); err != nil {
		log.Write(log.ERROR, "load Config {"+dbName+"} Parse Failed: ", err)
	}
	for idx := 0; idx < len(dbSlice); idx++ {
		orm.InitDBPoolSt().Set(dbSlice[idx].SKey, &dbSlice[idx])
	}
}
