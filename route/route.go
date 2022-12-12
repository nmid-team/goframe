package route

import (
	"github.com/gin-gonic/gin"
	"goframe/app/base/controller"
)

// RouteHome 主页
func RouteHome(parentRoute *gin.Engine) {
	parentRoute.GET("", controller.Welcome)
}

func RouteApi(parentRoute *gin.Engine) {
	parentRoute.GET("/healthcheck", controller.HealthCheck)
}
