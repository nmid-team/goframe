package response

import (
	"bytes"
	"encoding/json"
	"goframe/constv"
	"goframe/pkg/confer"
	"goframe/pkg/util"
	"net/http"

	"github.com/HughNian/nmid/pkg/logger"

	"github.com/gin-gonic/gin"
)

func UtilResponseReturnJsonNoP(c *gin.Context, code int, model interface{}, msg ...string) {
	UtilResponseReturnJsonWithMsg(c, code, getResponseMsg(code, msg...), model, false, true)
}

func UtilResponseReturnJson(c *gin.Context, code int, model interface{}, msg ...string) {
	UtilResponseReturnJsonWithMsg(c, code, getResponseMsg(code, msg...), model, true, true)
}

func UtilResponseReturnJsonNoPReal(c *gin.Context, code int, model interface{}, msg ...string) {
	UtilResponseReturnJsonWithMsg(c, code, getResponseMsg(code, msg...), model, false, false)
}

func UtilResponseReturnJsonReal(c *gin.Context, code int, model interface{}, msg ...string) {
	UtilResponseReturnJsonWithMsg(c, code, getResponseMsg(code, msg...), model, true, false)
}

func getResponseMsg(code int, msg ...string) (message string) {
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	if message == "" {
		message = confer.ConfigCodeGetMessage(code)
	}
	return
}

func UtilResponseReturnJsonWithMsg(c *gin.Context, code int, msg string, model interface{},
	callbackFlag bool, unifyCode bool) {
	if unifyCode && code == 0 {
		code = 1001
	}
	//if msg == "" {
	//	msg = confer.ConfigCodeGetMessage(code)
	//}
	// 放入返回的code码
	c.Set("result_code", code)
	// 判断是否存在error上下文
	if err, ok := c.Get("error"); ok {
		if err.(error) != nil {
			msg = err.(error).Error()
		}
	}
	rj := gin.H{
		"code":    code,
		"message": msg,
		"data":    model,
	}
	var callback string
	if callbackFlag {
		callback = c.Query("callback")
	}
	if util.UtilIsEmpty(callback) {
		// 根据code码返回不同的statusCode
		if status, ok := statusCode[code]; ok {
			c.JSON(status, rj)
		} else {
			c.JSON(http.StatusOK, rj)
		}
	} else {
		r, err := json.Marshal(rj)
		if err != nil {
			logger.Errorf("UtilResponseReturnJsonWithMsg json Marshal error", err)
		} else {
			c.String(http.StatusOK, "%s(%s)", callback, r)
		}
	}
}

func UtilResponseReturnJsonFailed(c *gin.Context, code int) {
	UtilResponseReturnJson(c, code, nil)
}

func UtilResponseReturnJsonSuccess(c *gin.Context, data interface{}) {
	UtilResponseReturnJson(c, 0, data)
}

func UtilResponseRedirect(c *gin.Context, url string) {
	c.Redirect(http.StatusMovedPermanently, url)
}

func utilResponseJSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

var statusCode = map[int]int{
	constv.CODE_ERROR_OK:                  http.StatusOK,
	constv.CODE_COMMON_OK:                 http.StatusOK,
	constv.CODE_COMMON_SERVER_BUSY:        http.StatusInternalServerError,
	constv.CODE_COMMON_PARAMS_INCOMPLETE:  http.StatusBadRequest,
	constv.CODE_COMMON_DATA_NOT_EXIST:     http.StatusBadRequest,
	constv.CODE_COMMON_DATA_ALREADY_EXIST: http.StatusBadRequest,
	constv.CODE_VICTORIA_METRICS_ERR:      http.StatusInternalServerError,
}
