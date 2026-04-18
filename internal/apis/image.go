package apis

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/orvice/objr/internal/upload"
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

	tmp, err := os.CreateTemp("", "objr-image-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	dst := tmp.Name()
	if err := tmp.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer func() {
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			slog.Error("remove file error", "err", err)
		}
	}()

	// Upload the file to specific dst.
	err = c.SaveUploadedFile(f, dst)
	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}

	ret, err := uploadService.UploadImage(c.Request.Context(), upload.Source{
		FilePath: dst,
		Filename: f.Filename,
	})
	if err != nil {
		c.JSON(500, gin.H{
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"message":      "success",
		"url":          ret.URL,
		"file_mine":    f.Header,
		"content_type": f.Header.Get("Content-Type"),
	})
}
