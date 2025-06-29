package apis

import (
	"net/http"

	"butterfly.orx.me/core/log"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/orvice/objr/internal/conf"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func Router(r *gin.Engine) {
	// cors
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true

	config.AllowHeaders = conf.Conf.CorsHeader
	r.Use(cors.New(config))

	r.Use(otelgin.Middleware("objr"))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.Use(auth)
	r.POST("/v1/image", uploadImage)
}

func auth(c *gin.Context) {
	if c.Request.Header.Get("Token") == conf.Conf.AuthToken {
		c.Next()
	} else {
		log.FromContext(c.Request.Context()).Error("auth failed", "token", c.Request.Header.Get("Token"))
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}
