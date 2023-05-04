package apis

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func Router(r *gin.Engine) {
	// cors
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true

	headers := os.Getenv("CORS_ALLOW_HEADERS")
	config.AllowHeaders = strings.Split(headers, ",")
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
	if c.Request.Header.Get("Token") == os.Getenv("AUTH_TOKEN") {
		c.Next()
	} else {
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}
