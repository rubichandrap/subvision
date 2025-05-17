package primitives

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func JSendSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   data,
	})
}

func JSendFail(c *gin.Context, data any, code int) {
	c.JSON(code, gin.H{
		"status": "fail",
		"data":   data,
	})
}

func JSendError(c *gin.Context, message string, code int, data any) {
	resp := gin.H{
		"status":  "error",
		"message": message,
	}
	if data != nil {
		resp["data"] = data
	}
	c.JSON(code, resp)
}
