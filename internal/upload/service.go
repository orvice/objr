package upload

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/orvice/objr/internal/object"
)

const DefaultAppPackageVersion = "nightly"

const latestAppPackageChannel = "latest"

type Uploader func(ctx context.Context, objectName string, filePath string, objectSize int64) (*object.UploadResult, error)

type Service struct {
	Uploader Uploader
	Now      func() time.Time
	NewUUID  func() (uuid.UUID, error)
}

func NewService() *Service {
	return &Service{
		Uploader: object.Upload,
		Now:      time.Now,
		NewUUID:  uuid.NewUUID,
	}
}

func NewServiceWithUploader(uploader Uploader) *Service {
	svc := NewService()
	svc.Uploader = uploader
	return svc
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

type Source struct {
	FilePath      string
	ContentBase64 string
	Filename      string
}

type ImageResult struct {
	Message     string `json:"message"`
	URL         string `json:"url"`
	ObjectKey   string `json:"object_key"`
	ContentType string `json:"content_type"`
}

type AppPackageResult struct {
	Message            string `json:"message"`
	DownloadURL        string `json:"download_url"`
	AppName            string `json:"app_name"`
	Version            string `json:"version"`
	ObjectKey          string `json:"object_key"`
	NightlyDownloadURL string `json:"nightly_download_url,omitempty"`
	NightlyObjectKey   string `json:"nightly_object_key,omitempty"`
	LatestDownloadURL  string `json:"latest_download_url,omitempty"`
	LatestObjectKey    string `json:"latest_object_key,omitempty"`
}

type AppPackageInput struct {
	Source  Source
	AppName string
	Version string
}

type resolvedSource struct {
	filePath string
	filename string
	size     int64
	cleanup  func()
}

func (s *Service) UploadImage(ctx context.Context, source Source) (*ImageResult, error) {
	resolved, err := resolveSource(source)
	if err != nil {
		return nil, err
	}
	defer resolved.cleanup()

	filename := CleanUploadedFilename(resolved.filename)
	if filename == "" {
		return nil, validationError("file name is required")
	}

	id, err := s.newUUID()
	if err != nil {
		return nil, err
	}

	objectName := ImageObjectName(filename, s.now(), id)
	ret, err := s.upload(ctx, objectName, resolved.filePath, resolved.size)
	if err != nil {
		return nil, err
	}

	return &ImageResult{
		Message:     "success",
		URL:         ret.URL,
		ObjectKey:   objectName,
		ContentType: detectContentType(resolved.filePath),
	}, nil
}

func (s *Service) UploadAppPackage(ctx context.Context, input AppPackageInput) (*AppPackageResult, error) {
	resolved, err := resolveSource(input.Source)
	if err != nil {
		return nil, err
	}
	defer resolved.cleanup()

	appName := SanitizePathSegment(input.AppName)
	if appName == "" {
		return nil, validationError("app_name is required")
	}

	if !IsAllowedAppPackage(resolved.filename) {
		return nil, validationError("file must be .apk or .aab")
	}

	version := SanitizePathSegment(NormalizeAppPackageVersion(input.Version))
	if version == "" {
		return nil, validationError("version is invalid")
	}

	filename := SanitizePathSegment(CleanUploadedFilename(resolved.filename))
	if filename == "" {
		return nil, validationError("file name is invalid")
	}

	id, err := s.newUUID()
	if err != nil {
		return nil, err
	}

	objectName := AppPackageObjectName(appName, version, filename, s.now(), id)
	ret, err := s.upload(ctx, objectName, resolved.filePath, resolved.size)
	if err != nil {
		return nil, err
	}

	result := &AppPackageResult{
		Message:     "success",
		DownloadURL: ret.URL,
		AppName:     appName,
		Version:     version,
		ObjectKey:   objectName,
	}

	if version == DefaultAppPackageVersion {
		aliasObjectName := AppPackageAliasObjectName(appName, DefaultAppPackageVersion, filename)
		aliasRet, err := s.upload(ctx, aliasObjectName, resolved.filePath, resolved.size)
		if err != nil {
			return nil, err
		}
		result.NightlyDownloadURL = aliasRet.URL
		result.NightlyObjectKey = aliasObjectName
	} else {
		aliasObjectName := AppPackageAliasObjectName(appName, latestAppPackageChannel, filename)
		aliasRet, err := s.upload(ctx, aliasObjectName, resolved.filePath, resolved.size)
		if err != nil {
			return nil, err
		}
		result.LatestDownloadURL = aliasRet.URL
		result.LatestObjectKey = aliasObjectName
	}

	return result, nil
}

func NormalizeAppPackageVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return DefaultAppPackageVersion
	}
	return version
}

func IsAllowedAppPackage(filename string) bool {
	switch strings.ToLower(path.Ext(CleanUploadedFilename(filename))) {
	case ".apk", ".aab":
		return true
	default:
		return false
	}
}

func CleanUploadedFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = path.Base(filename)
	if filename == "." || filename == "/" {
		return ""
	}
	return filename
}

func SanitizePathSegment(value string) string {
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

func ImageObjectName(filename string, now time.Time, id uuid.UUID) string {
	return fmt.Sprintf("images/%d/%d/%d/%s-%s",
		now.Year(),
		now.Month(),
		now.Day(),
		id.String(),
		filename)
}

func AppPackageObjectName(appName string, version string, filename string, now time.Time, id uuid.UUID) string {
	return fmt.Sprintf("apps/%s/%s/%d/%d/%d/%s-%s",
		appName,
		version,
		now.Year(),
		now.Month(),
		now.Day(),
		id.String(),
		filename)
}

func AppPackageAliasObjectName(appName string, channel string, filename string) string {
	return fmt.Sprintf("apps/%s/%s/app%s",
		appName,
		channel,
		strings.ToLower(path.Ext(filename)))
}

func resolveSource(source Source) (*resolvedSource, error) {
	filePath := strings.TrimSpace(source.FilePath)
	contentBase64 := strings.TrimSpace(source.ContentBase64)

	if filePath == "" && contentBase64 == "" {
		return nil, validationError("file_path or content_base64 is required")
	}
	if filePath != "" && contentBase64 != "" {
		return nil, validationError("provide either file_path or content_base64, not both")
	}

	if filePath != "" {
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, validationError("file_path must be a file")
		}

		filename := strings.TrimSpace(source.Filename)
		if filename == "" {
			filename = info.Name()
		}
		return &resolvedSource{
			filePath: filePath,
			filename: filename,
			size:     info.Size(),
			cleanup:  func() {},
		}, nil
	}

	filename := CleanUploadedFilename(source.Filename)
	if filename == "" {
		return nil, validationError("filename is required when using content_base64")
	}

	content, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp("", "objr-upload-*"+strings.ToLower(path.Ext(filename)))
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return nil, err
	}

	return &resolvedSource{
		filePath: tmpPath,
		filename: filename,
		size:     int64(len(content)),
		cleanup:  cleanup,
	}, nil
}

func detectContentType(filePath string) string {
	mtype, err := mimetype.DetectFile(filePath)
	if err != nil {
		return ""
	}
	return mtype.String()
}

func validationError(message string) error {
	return &ValidationError{Message: message}
}

func (s *Service) upload(ctx context.Context, objectName string, filePath string, objectSize int64) (*object.UploadResult, error) {
	uploader := s.Uploader
	if uploader == nil {
		uploader = object.Upload
	}
	return uploader(ctx, objectName, filePath, objectSize)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) newUUID() (uuid.UUID, error) {
	if s.NewUUID != nil {
		return s.NewUUID()
	}
	return uuid.NewUUID()
}
