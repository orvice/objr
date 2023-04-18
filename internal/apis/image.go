package apis

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/orvice/objr/internal/object"
	"golang.org/x/exp/slog"
)

func uploadImage(c *gin.Context) {
	// single file
	f, err := c.FormFile("image")
	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}
	slog.Info("upload file", "file", f.Filename, "size", f.Size)

	// gen object name with date
	now := time.Now()
	objectName := fmt.Sprintf("images/%d/%d/%d/%d/%s", now.Year(), now.Month(), now.Day(), now.Unix(), f.Filename)
	dst := fmt.Sprintf("/tmp/%s", f.Filename)
	// Upload the file to specific dst.
	err = c.SaveUploadedFile(f, dst)
	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}
	ret, err := object.Upload(c.Request.Context(), objectName, dst, f.Size)
	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}
	// clean dst
	err = os.Remove(dst)
	if err != nil {
		slog.Error("remove file error", "err", err)
	}
	c.JSON(200, gin.H{
		"message": "success",
		"url":     ret.URL,
	})
}
