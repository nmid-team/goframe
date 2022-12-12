package confer

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var v *viper.Viper
var globalConfig *Server
var mutex sync.RWMutex

func Init(configURL string) (err error) {
	v = viper.New()
	v.SetConfigFile(configURL)
	//v.AutomaticEnv()
	//err = v.ReadInConfig()
	confContent, err := ioutil.ReadFile(configURL)
	if err != nil {
		log.Fatal(fmt.Sprintf("Read config file fail: %s", err.Error()))
	}
	//Replace environment variables
	err = v.ReadConfig(strings.NewReader(os.ExpandEnv(string(confContent))))
	if err != nil {
		err = fmt.Errorf("Fatal error config file: %w", err)
		return
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Println("config file changed:", e.Name)
		globalConfig.Lock()
		defer globalConfig.Unlock()
		if err := v.Unmarshal(&globalConfig); err != nil {
			log.Println(err)
		}
	})
	if err := v.Unmarshal(&globalConfig); err != nil {
		return err
	}
	return changeDataByEnv()
}

func changeDataByEnv() (err error) {
	// 一些动态的值，根据环境变量获取
	if redis := os.Getenv(globalConfig.Redis.Address); len(redis) > 0 {
		globalConfig.Redis.Address = redis
	}
	if mysqlDbname := os.Getenv(globalConfig.Mysql.DBName); len(mysqlDbname) > 0 {
		globalConfig.Mysql.DBName = mysqlDbname
	}
	if mysqlWriteAddr := os.Getenv(globalConfig.Mysql.Write.Host); len(mysqlWriteAddr) > 0 {
		globalConfig.Mysql.Write.Host = mysqlWriteAddr
	}

	// 处理mysql地址
	var host = "127.0.0.1"
	var port = "3306"
	if len(globalConfig.Mysql.Write.Host) > 0 {
		host, port, err = net.SplitHostPort(globalConfig.Mysql.Write.Host)
		if err != nil {
			err = fmt.Errorf("mysql host port is wrong :%w,%s", err, globalConfig.Mysql.Write.Host)
			return
		}
	}
	globalConfig.Mysql.Write.Host = host
	portInt, _ := strconv.Atoi(port)
	globalConfig.Mysql.Write.Port = portInt

	if mysqlWriteUser := os.Getenv(globalConfig.Mysql.Write.User); len(mysqlWriteUser) > 0 {
		globalConfig.Mysql.Write.User = mysqlWriteUser
	}
	if mysqlWritePwd := os.Getenv(globalConfig.Mysql.Write.Password); len(mysqlWritePwd) > 0 {
		globalConfig.Mysql.Write.Password = mysqlWritePwd
	}
	globalConfig.Mysql.Write.DBName = globalConfig.Mysql.DBName
	globalConfig.Mysql.Write.Prefix = globalConfig.Mysql.Prefix

	if logRedisHost := os.Getenv(globalConfig.Log.Redis.Host); len(logRedisHost) > 0 {
		globalConfig.Log.Redis.Host = logRedisHost
	}
	if logAppName := os.Getenv(globalConfig.Log.App.AppName); len(logAppName) > 0 {
		globalConfig.Log.App.AppName = logAppName
	}
	if logAppVersion := os.Getenv(globalConfig.Log.App.AppVersion); len(logAppVersion) > 0 {
		globalConfig.Log.App.AppVersion = logAppVersion
	}
	if logAppSubOrgLanguage := os.Getenv(globalConfig.Log.App.Language); len(logAppSubOrgLanguage) > 0 {
		globalConfig.Log.App.Language = logAppSubOrgLanguage
	}
	return
}

func GetGlobalConfig() *Server {
	mutex.RLock()
	defer mutex.RUnlock()
	return globalConfig
}
