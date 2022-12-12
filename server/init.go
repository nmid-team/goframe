package server

import (
	"fmt"
	"github.com/HughNian/nmid/pkg/logger"
	"goframe/pkg/confer"
	"goframe/pkg/initer"
	"goframe/pkg/mysql"
	"runtime"
	"time"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/urfave/cli"
)

func InitService(c *cli.Context) error {
	//环境初始化
	configRuntime()
	// 初始化配置文件及内部服务
	err := initer.ConfigAndBase(c.String("c"))
	if err != nil {
		logger.Fatal(fmt.Sprintf("init ConfigAndBase err : %v", err))
	}
	// 初始化外部依赖服务
	err = initer.OutSideResource()
	if err != nil {
		logger.Fatal(fmt.Sprintf("init OutSideResource err : %v", err))
	}
	if !confer.ConfigEnvIsDev() && confer.GetGlobalConfig().Mysql.Enabled {
		sqlMigrate()
	}
	return nil
}

func configRuntime() {
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	now := time.Now().String()
	fmt.Printf("Running time is %s\n", now)
	fmt.Printf("Running with %d CPUs\n", numCPU)
}

func sqlMigrate() {
	// docker 环境下，根据docker file 配置，sql文件在统计db目录下
	migrations := &migrate.FileMigrationSource{
		Dir: "./db",
	}
	Orm := mysql.NewDaoMysql().GetOrm()
	sqlDb, err := Orm.DB.DB()
	if err != nil {
		logger.Errorf("sqlMigrate Orm.DB.DB() err ", err)
		return
	}
	code, err := migrate.Exec(sqlDb, "mysql", migrations, migrate.Up)
	if err != nil {
		logger.Errorf("sqlMigrate err ", err)
		return
	}
	logger.Errorf("sqlMigrate code is : %d", code)
}
