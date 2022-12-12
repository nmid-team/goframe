package mysql

import (
	"errors"
	"fmt"
	"github.com/HughNian/nmid/pkg/logger"
	"goframe/pkg/confer"
	"log"
	"math/rand"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type DaoMysql struct {
	TableName string
}

func NewDaoMysql() *DaoMysql {
	return &DaoMysql{}
}

type MysqlConnection struct {
	*gorm.DB
	IsRead bool
}

//func (p MysqlConnection) Close() {
//	if p.DB != nil {
//		_ = p.DB.Close()
//	}
//}

func (p MysqlConnection) Put() {
	//db database sql inner put
}

var (
	mysqlReadPool  MysqlConnection
	mysqlWritePool MysqlConnection
)

func InitMysqlPool(conf confer.Mysql, isRead bool) (err error) {
	if isRead {
		mysqlReadPool.DB, err = initDb(conf, isRead)
		mysqlReadPool.IsRead = isRead
	} else {
		mysqlWritePool.DB, err = initDb(conf, isRead)
		mysqlWritePool.IsRead = isRead
	}
	if err != nil {
		err = errors.New(fmt.Sprintf("initMysqlPool isread:%v ,error: %v", isRead, err))
		return
	}
	if isRead {
		sqlDB, err := mysqlReadPool.DB.DB()
		if err != nil {
			err = errors.New(fmt.Sprintf("initMysqlPool isread:%v ,error: %v", isRead, err))
			return err
		}
		sqlDB.SetMaxIdleConns(conf.Pool.PoolMinCap)                       // 空闲链接
		sqlDB.SetMaxOpenConns(conf.Pool.PoolMaxCap)                       // 最大链接
		sqlDB.SetConnMaxLifetime(conf.Pool.PoolIdleTimeout * time.Second) // 最大空闲时间
	} else {
		sqlDB, err := mysqlWritePool.DB.DB()
		if err != nil {
			err = errors.New(fmt.Sprintf("initMysqlPool isread:%v ,error: %v", isRead, err))
			return err
		}
		sqlDB.SetMaxIdleConns(conf.Pool.PoolMinCap)                       // 空闲链接
		sqlDB.SetMaxOpenConns(conf.Pool.PoolMaxCap)                       // 最大链接
		sqlDB.SetConnMaxLifetime(conf.Pool.PoolIdleTimeout * time.Second) // 最大空闲时间
	}
	return
}

func initDb(conf confer.Mysql, isRead bool) (resultDb *gorm.DB, err error) {
	var dbConfig confer.DBBase
	if isRead && len(conf.Reads) > 0 {
		rand.Seed(time.Now().UnixNano())
		dbConfig = conf.Reads[rand.Intn(len(conf.Reads)-1)]
	} else {
		dbConfig = conf.Write
	}
	// 判断配置可用性
	if dbConfig.Host == "" || dbConfig.DBName == "" {
		err = errors.New("dbConfig is null")
		return
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4,utf8&parseTime=True&loc=Local", dbConfig.User,
		dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.DBName)
	config := &gorm.Config{
		SkipDefaultTransaction: true,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   conf.Prefix, // 表名前缀，`User`表为`t_users`
			SingularTable: true,        // 使用单数表名，启用该选项后，`User` 表将是`user`
		},
	}
	if confer.ConfigEnvIsDev() {
		newLogger := glogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			glogger.Config{
				SlowThreshold: time.Second,  // 慢 SQL 阈值
				LogLevel:      glogger.Info, // Log level
				Colorful:      true,         // 禁用彩色打印
			},
		)
		config.Logger = newLogger
	}
	resultDb, err = gorm.Open(mysql.Open(dsn), config)
	if err != nil {
		logger.Errorf("connect mysql error", err)
		return resultDb, err
	}

	return resultDb, err
}

func initMysqlPoolConnection(isRead bool) (conn MysqlConnection) {
	if isRead {
		conn = mysqlReadPool
	} else {
		conn = mysqlWritePool
	}
	return
}

func (p *DaoMysql) GetReadOrm() MysqlConnection {
	return p.getOrm(true)
}

func (p *DaoMysql) GetWriteOrm() MysqlConnection {
	return p.getOrm(false)
}

func (p *DaoMysql) GetOrm() MysqlConnection {
	return p.getOrm(false)
}

func (p *DaoMysql) getOrm(isRead bool) MysqlConnection {
	return initMysqlPoolConnection(isRead)
}
