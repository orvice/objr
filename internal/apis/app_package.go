package apis

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/orvice/objr/internal/upload"
	"golang.org/x/exp/slog"
)

func uploadAppPackage(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "file is required",
		})
		return
	}

	tmp, err := os.CreateTemp("", "objr-app-package-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	defer func() {
		if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
			slog.Error("remove app package temp file error", "err", err)
		}
	}()

	if err := c.SaveUploadedFile(f, tmpPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	ret, err := uploadService.UploadAppPackage(c.Request.Context(), upload.AppPackageInput{
		Source: upload.Source{
			FilePath: tmpPath,
			Filename: f.Filename,
		},
		AppName: c.PostForm("app_name"),
		Version: c.PostForm("version"),
	})
	if err != nil {
		status := http.StatusInternalServerError
		if upload.IsValidationError(err) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{
			"message": err.Error(),
		})
		return
	}

	slog.Info("upload app package",
		"file", f.Filename,
		"size", f.Size,
		"appName", ret.AppName,
		"version", ret.Version,
		"objectName", ret.ObjectKey)

	response := gin.H{
		"message":      ret.Message,
		"download_url": ret.DownloadURL,
		"app_name":     ret.AppName,
		"version":      ret.Version,
		"object_key":   ret.ObjectKey,
	}
	if strings.TrimSpace(ret.NightlyDownloadURL) != "" {
		response["nightly_download_url"] = ret.NightlyDownloadURL
		response["nightly_object_key"] = ret.NightlyObjectKey
	}
	if strings.TrimSpace(ret.LatestDownloadURL) != "" {
		response["latest_download_url"] = ret.LatestDownloadURL
		response["latest_object_key"] = ret.LatestObjectKey
	}

	c.JSON(http.StatusOK, response)
}
