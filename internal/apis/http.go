package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func Router() {
	r := gin.Default()
	r.Use(otelgin.Middleware("objr"))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/v1/image", uploadImage)
	_ = r.Run(":8080")
}
