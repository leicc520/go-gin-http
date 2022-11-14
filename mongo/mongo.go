package mongo

import (
	"context"
	"sync"
	"time"

	"git.ziniao.com/webscraper/go-orm/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoSt struct {
	Host         string                 `yaml:"host"`
	DataBaseName string                 `yaml:"databaseName"`
	MaxPoolConns uint64                 `yaml:"maxPoolConns"`
	onceSymc     sync.Once              `yaml:"-"`
	clientOption *options.ClientOptions `yaml:"-"`
}

// 初始化链接mongodb 初始化只要执行一次即可
func (s *MongoSt) Init() {
	s.onceSymc.Do(func() {
		cliOptions := options.Client().ApplyURI(s.Host)
		cliOptions.SetConnectTimeout(connectTimeOut)
		if s.MaxPoolConns <= 1 || s.MaxPoolConns > 10240 {
			cliOptions.SetMaxPoolSize(s.MaxPoolConns)
		}
		s.clientOption = cliOptions
	})
}

// 执行ping一下服务器看看链接i情况
func (s MongoSt) Ping(client *mongo.Client, database *mongo.Database) (*mongo.Client, *mongo.Database, bool) {
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Write(-1, "mongo Ping 失败", err)
		//ping如果失败的话可以重连机制
		if client, database, err = s.Client(time.Second * 30); err != nil {
			return nil, nil, false
		}
	}
	return client, database, true
}

// 获取一个mongo请求的业务链接
func (s *MongoSt) Client(timeOut time.Duration) (*mongo.Client, *mongo.Database, error) {
	s.Init() //初始化内容
	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()
	cli, err := mongo.Connect(ctx, s.clientOption)
	if err != nil { //链接失败报错
		log.Write(log.ERROR, "mongo 链接失败", err)
		return nil, nil, err
	}
	database := cli.Database(s.DataBaseName)
	return cli, database, err
}
