package confer

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Server struct {
	App   App   `mapstructure:"app" json:"app" yaml:"app"`
	Code  Code  `mapstructure:"code" json:"code" yaml:"code"`
	Nmid  Nmid  `mapstructure:"nmid" json:"nmid" yaml:"nmid"`
	Redis Redis `mapstructure:"redis" json:"redis" yaml:"redis"`
	Gzip  Gzip  `mapstructure:"gzip" json:"gzip" yaml:"gzip"`
	Mysql Mysql `mapstructure:"mysql" json:"mysql" yaml:"mysql"`
	Log   Log   `mapstructure:"log" json:"log" yaml:"log"`
	sync.RWMutex
}

type App map[string]interface{}

type Code map[string]interface{}

type Nmid map[string]interface{}

type Redis struct {
	Enabled                bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Address                string `mapstructure:"address" json:"address" yaml:"address"`
	Prefix                 string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
	Expire                 int    `mapstructure:"expire" json:"expire" yaml:"expire"`
	ConnectTimeout         int    `mapstructure:"connect-timeout" json:"connectTimeout" yaml:"connect-timeout"`
	ReadTimeout            int    `mapstructure:"read-timeout" json:"readTimeout" yaml:"read-timeout"`
	WriteTimeout           int    `mapstructure:"write-timeout" json:"writeTimeout" yaml:"write-timeout"`
	PoolMaxIdel            int    `mapstructure:"pool-max-idel" json:"poolMaxIdel" yaml:"pool-max-idel"`
	PoolMaxActive          int    `mapstructure:"pool-max-active" json:"poolMaxActive" yaml:"pool-max-active"`
	PoolMinActive          int    `mapstructure:"pool-min-active" json:"poolMinActive" yaml:"pool-min-active"`
	PoolIdleTimeout        int    `mapstructure:"pool-idle-timeout" json:"poolIdleTimeout" yaml:"pool-idle-timeout"`
	ClusterUpdateHeartbeat int    `mapstructure:"cluster-update-heartbeat" json:"clusterUpdateHeartbeat" yaml:"cluster-update-heartbeat"`
	Password               string `mapstructure:"password" json:"password" yaml:"password"`
}

type Gzip struct {
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Level   int  `mapstructure:"level" json:"level" yaml:"level"`
}

type Mysql struct {
	Enabled bool     `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	DBName  string   `mapstructure:"dbname" json:"dbName" yaml:"dbname"`
	Prefix  string   `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
	Pool    DBPool   `mapstructure:"pool" json:"pool" yaml:"pool"`
	Write   DBBase   `mapstructure:"write" json:"write" yaml:"write"`
	Reads   []DBBase `mapstructure:"reads" json:"reads" yaml:"reads"`
}

type DBPool struct {
	PoolMinCap      int           `mapstructure:"pool-min-cap" json:"poolMinCap" yaml:"pool-min-cap"`
	PoolExCap       int           `mapstructure:"pool-ex-cap" json:"poolExCap" yaml:"pool-ex-cap"`
	PoolMaxCap      int           `mapstructure:"pool-max-cap" json:"pool-max-cap" yaml:"pool-max-cap"`
	PoolIdleTimeout time.Duration `mapstructure:"pool-idle-timeout" json:"poolIdleTimeout" yaml:"pool-idle-timeout"`
	PoolWaitCount   int64         `mapstructure:"pool-wait-count" json:"poolWaitCount" yaml:"pool-wait-count"`
	PoolWaitTimeout time.Duration `mapstructure:"pool-wai-timeout" json:"poolWaitTimeout" yaml:"pool-wai-timeout"`
}

type DBBase struct {
	Host     string `mapstructure:"host" json:"host" yaml:"host"`
	Port     int    `mapstructure:"port" json:"port" yaml:"port"`
	User     string `mapstructure:"user" json:"user" yaml:"user"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	DBName   string `json:"-"`
	Prefix   string `json:"-"`
}

type Log struct {
	Enabled bool         `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	OutPut  string       `mapstructure:"out-put" json:"outPut" yaml:"out-put"`
	Debug   bool         `mapstructure:"debug" json:"debug" yaml:"debug"`
	Key     string       `mapstructure:"key" json:"key" yaml:"key"`
	Level   logrus.Level `mapstructure:"level" json:"level" yaml:"level"`
	Redis   struct {
		Host string
		Port int
	}
	App struct {
		AppName    string `mapstructure:"app-name" json:"appName" yaml:"app-name"`
		AppVersion string `mapstructure:"app-version" json:"appVersion" yaml:"app-version"`
		Language   string `mapstructure:"language" json:"language" yaml:"language"`
	} `mapstructure:"app" json:"app" yaml:"app"`
}
