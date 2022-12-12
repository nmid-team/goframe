package gin

import (
	"fmt"
	"github.com/HughNian/nmid/pkg/logger"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				go logger.Errorf("panic recovered;"+";stacktrace from panic:\n"+string(debug.Stack()), fmt.Errorf("%v", err))
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
