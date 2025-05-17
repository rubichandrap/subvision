package handler

import (
	"github.com/gin-gonic/gin"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

func RegisterTusd(r *gin.Engine, clientURL string, handler *tusd.Handler) {
	r.Any("/files/*path", func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	})
}
