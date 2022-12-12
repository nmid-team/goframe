package server

import (
	"goframe/middleware"
	"goframe/pkg/confer"
	"goframe/pkg/gin"
	"goframe/route"
	"strconv"

	"github.com/gin-contrib/gzip"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var httpPort int

func RunHTTP() {
	println("RunHttp Server.")
	r := gin.NewGin()
	// 跨域
	r.Use(middleware.Cors())
	// gzip压缩
	if confer.GetGlobalConfig().Gzip.Enabled {
		r.Use(gzip.Gzip(confer.GetGlobalConfig().Gzip.Level))
	}
	httpPort = confer.ConfigAppGetInt("port", 80)
	println("|- http start at:", httpPort)
	portStr := ":" + strconv.Itoa(httpPort)
	if confer.ConfigEnvIsDev() {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	route.RouteHome(r)
	route.RouteApi(r)
	gin.ListenHttp(portStr, r, 10)
}
