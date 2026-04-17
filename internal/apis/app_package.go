package apis

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/orvice/objr/internal/object"
	"golang.org/x/exp/slog"
)

const defaultAppPackageVersion = "nightly"

type appPackageUploader func(ctx context.Context, objectName string, filePath string, objectSize int64) (*object.UploadResult, error)

var uploadObject appPackageUploader = object.Upload

func uploadAppPackage(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "file is required",
		})
		return
	}

	appName := sanitizePathSegment(c.PostForm("app_name"))
	if appName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "app_name is required",
		})
		return
	}

	if !isAllowedAppPackage(f.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "file must be .apk or .aab",
		})
		return
	}

	version := sanitizePathSegment(normalizeAppPackageVersion(c.PostForm("version")))
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "version is invalid",
		})
		return
	}

	filename := sanitizePathSegment(cleanUploadedFilename(f.Filename))
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "file name is invalid",
		})
		return
	}

	id, err := uuid.NewUUID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	objectName := appPackageObjectName(appName, version, filename, time.Now(), id)
	tmp, err := os.CreateTemp("", "objr-app-package-*"+strings.ToLower(path.Ext(filename)))
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

	slog.Info("upload app package",
		"file", f.Filename,
		"size", f.Size,
		"appName", appName,
		"version", version,
		"objectName", objectName)

	ret, err := uploadObject(c.Request.Context(), objectName, tmpPath, f.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	response := gin.H{
		"message":      "success",
		"download_url": ret.URL,
		"app_name":     appName,
		"version":      version,
		"object_key":   objectName,
	}

	if version == defaultAppPackageVersion {
		aliasObjectName := appPackageAliasObjectName(appName, defaultAppPackageVersion, filename)
		aliasRet, err := uploadObject(c.Request.Context(), aliasObjectName, tmpPath, f.Size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}
		response["nightly_download_url"] = aliasRet.URL
		response["nightly_object_key"] = aliasObjectName
	} else {
		aliasObjectName := appPackageAliasObjectName(appName, "latest", filename)
		aliasRet, err := uploadObject(c.Request.Context(), aliasObjectName, tmpPath, f.Size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}
		response["latest_download_url"] = aliasRet.URL
		response["latest_object_key"] = aliasObjectName
	}

	c.JSON(http.StatusOK, response)
}

func normalizeAppPackageVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return defaultAppPackageVersion
	}
	return version
}

func isAllowedAppPackage(filename string) bool {
	switch strings.ToLower(path.Ext(cleanUploadedFilename(filename))) {
	case ".apk", ".aab":
		return true
	default:
		return false
	}
}

func cleanUploadedFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "\\", "/")
	return path.Base(filename)
}

func sanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)

	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}

	return strings.Trim(b.String(), ".-")
}

func appPackageObjectName(appName string, version string, filename string, now time.Time, id uuid.UUID) string {
	return fmt.Sprintf("apps/%s/%s/%d/%d/%d/%s-%s",
		appName,
		version,
		now.Year(),
		now.Month(),
		now.Day(),
		id.String(),
		filename)
}

func appPackageAliasObjectName(appName string, channel string, filename string) string {
	return fmt.Sprintf("apps/%s/%s/app%s",
		appName,
		channel,
		strings.ToLower(path.Ext(filename)))
}
