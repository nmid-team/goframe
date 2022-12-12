package initer

import (
	"goframe/pkg/confer"
	"goframe/pkg/mysql"
	"goframe/pkg/redis"
)

func ConfigAndBase(configURL string) (err error) {
	err = confer.Init(configURL)
	if err != nil {
		return
	}
	confer.ConfigCodeInit()
	if confer.GetGlobalConfig().Log.Enabled {
	}
	return
}

func OutSideResource() (err error) {

	if confer.GetGlobalConfig().Redis.Enabled {
		redis.InitRedis(confer.GetGlobalConfig().Redis)
	}
	if confer.GetGlobalConfig().Mysql.Enabled {
		err = initMysql()
		if err != nil {
			return err
		}
	}
	return nil
}

// 初始化mysql连接池
func initMysql() (err error) {
	err = mysql.InitMysqlPool(confer.GetGlobalConfig().Mysql, false) // 初始化写库，一个
	return
}
