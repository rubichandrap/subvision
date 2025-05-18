package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

func RegisterTusd(r *gin.Engine, clientURL string, handler *tusd.Handler) {
	filesHanler := http.StripPrefix("/files/", handler)
	filesRootHandler := http.StripPrefix("/files", handler)

	r.Any("/files/*path", gin.WrapH(filesHanler))
	r.Any("/files", gin.WrapH(filesRootHandler))
}
