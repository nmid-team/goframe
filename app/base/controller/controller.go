package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type result struct {
	State   int       `json:"state"`
	Message string    `json:"msg"`
	Data    *struct{} `json:"data,omitempty"`
}

func HealthCheck(c *gin.Context) {
	res := result{
		State:   200,
		Message: "success",
	}

	c.JSON(http.StatusOK, res)
}
