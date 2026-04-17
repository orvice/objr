package apis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/orvice/objr/internal/conf"
	"github.com/orvice/objr/internal/object"
)

func TestNormalizeAppPackageVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "missing", version: "", want: defaultAppPackageVersion},
		{name: "blank", version: " \t ", want: defaultAppPackageVersion},
		{name: "provided", version: " 1.2.3 ", want: "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeAppPackageVersion(tt.version); got != tt.want {
				t.Fatalf("normalizeAppPackageVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsAllowedAppPackage(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{name: "apk", filename: "demo.apk", want: true},
		{name: "aab", filename: "demo.aab", want: true},
		{name: "uppercase", filename: "demo.APK", want: true},
		{name: "unsupported", filename: "demo.zip", want: false},
		{name: "path stripped", filename: `build\release\demo.aab`, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAllowedAppPackage(tt.filename); got != tt.want {
				t.Fatalf("isAllowedAppPackage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizePathSegment(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "trims and replaces separators", value: " demo app/../release ", want: "demo-app-..-release"},
		{name: "keeps safe characters", value: "demo_app-1.2.3", want: "demo_app-1.2.3"},
		{name: "blank after cleanup", value: " / ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizePathSegment(tt.value); got != tt.want {
				t.Fatalf("sanitizePathSegment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAppPackageObjectName(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	now := time.Date(2026, time.April, 18, 10, 30, 0, 0, time.UTC)

	got := appPackageObjectName("demo-app", "1.2.3", "demo.apk", now, id)
	want := "apps/demo-app/1.2.3/2026/4/18/11111111-2222-3333-4444-555555555555-demo.apk"
	if got != want {
		t.Fatalf("appPackageObjectName() = %q, want %q", got, want)
	}
}

func TestAppPackageAliasObjectName(t *testing.T) {
	tests := []struct {
		name     string
		channel  string
		filename string
		want     string
	}{
		{name: "nightly apk", channel: "nightly", filename: "demo.apk", want: "apps/demo-app/nightly/app.apk"},
		{name: "nightly aab", channel: "nightly", filename: "demo.aab", want: "apps/demo-app/nightly/app.aab"},
		{name: "latest apk", channel: "latest", filename: "demo.apk", want: "apps/demo-app/latest/app.apk"},
		{name: "latest uppercase", channel: "latest", filename: "demo.APK", want: "apps/demo-app/latest/app.apk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appPackageAliasObjectName("demo-app", tt.channel, tt.filename); got != tt.want {
				t.Fatalf("appPackageAliasObjectName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUploadAppPackageSuccess(t *testing.T) {
	router := newAppPackageTestRouter(t)

	var capturedObjectNames []string
	stubUploadObject(t, func(ctx context.Context, objectName string, filePath string, objectSize int64) (*object.UploadResult, error) {
		if _, err := os.Stat(filePath); err != nil {
			t.Fatalf("expected temp file to exist during upload: %v", err)
		}
		if objectSize == 0 {
			t.Fatal("expected non-zero object size")
		}
		capturedObjectNames = append(capturedObjectNames, objectName)
		return &object.UploadResult{URL: "https://cdn.example.com/" + objectName}, nil
	})

	req := newMultipartRequest(t, "/v1/app-package", map[string]string{
		"app_name": "demo-app",
	}, "file", "demo.apk", []byte("fake apk contents"))
	req.Header.Set("Token", "test-token")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body["message"] != "success" {
		t.Fatalf("message = %q, want success", body["message"])
	}
	if body["app_name"] != "demo-app" {
		t.Fatalf("app_name = %q, want demo-app", body["app_name"])
	}
	if body["version"] != defaultAppPackageVersion {
		t.Fatalf("version = %q, want %q", body["version"], defaultAppPackageVersion)
	}
	if len(capturedObjectNames) != 2 {
		t.Fatalf("upload calls = %d, want 2", len(capturedObjectNames))
	}
	historicalObjectName := capturedObjectNames[0]
	nightlyObjectName := capturedObjectNames[1]
	if body["object_key"] != historicalObjectName {
		t.Fatalf("object_key = %q, want %q", body["object_key"], historicalObjectName)
	}
	if !strings.HasPrefix(historicalObjectName, "apps/demo-app/nightly/") {
		t.Fatalf("object key %q does not include expected app/version prefix", historicalObjectName)
	}
	if !strings.HasSuffix(historicalObjectName, "-demo.apk") {
		t.Fatalf("object key %q does not include original file name", historicalObjectName)
	}
	if body["download_url"] != "https://cdn.example.com/"+historicalObjectName {
		t.Fatalf("download_url = %q, want CDN URL plus object key", body["download_url"])
	}
	if nightlyObjectName != "apps/demo-app/nightly/app.apk" {
		t.Fatalf("nightly object key = %q, want stable APK alias", nightlyObjectName)
	}
	if body["nightly_object_key"] != nightlyObjectName {
		t.Fatalf("nightly_object_key = %q, want %q", body["nightly_object_key"], nightlyObjectName)
	}
	if body["nightly_download_url"] != "https://cdn.example.com/"+nightlyObjectName {
		t.Fatalf("nightly_download_url = %q, want CDN URL plus nightly object key", body["nightly_download_url"])
	}
	if _, ok := body["latest_download_url"]; ok {
		t.Fatal("latest_download_url was returned for nightly upload")
	}
	if _, ok := body["latest_object_key"]; ok {
		t.Fatal("latest_object_key was returned for nightly upload")
	}
}

func TestUploadAppPackageNightlyAABUsesStableAlias(t *testing.T) {
	router := newAppPackageTestRouter(t)

	var capturedObjectNames []string
	stubUploadObject(t, func(ctx context.Context, objectName string, filePath string, objectSize int64) (*object.UploadResult, error) {
		capturedObjectNames = append(capturedObjectNames, objectName)
		return &object.UploadResult{URL: "https://cdn.example.com/" + objectName}, nil
	})

	req := newMultipartRequest(t, "/v1/app-package", map[string]string{
		"app_name": "demo-app",
		"version":  " ",
	}, "file", "demo.aab", []byte("fake aab contents"))
	req.Header.Set("Token", "test-token")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(capturedObjectNames) != 2 {
		t.Fatalf("upload calls = %d, want 2", len(capturedObjectNames))
	}
	if capturedObjectNames[1] != "apps/demo-app/nightly/app.aab" {
		t.Fatalf("nightly object key = %q, want stable AAB alias", capturedObjectNames[1])
	}
}

func TestUploadAppPackageNonNightlyWritesLatestAlias(t *testing.T) {
	router := newAppPackageTestRouter(t)

	var capturedObjectNames []string
	stubUploadObject(t, func(ctx context.Context, objectName string, filePath string, objectSize int64) (*object.UploadResult, error) {
		capturedObjectNames = append(capturedObjectNames, objectName)
		return &object.UploadResult{URL: "https://cdn.example.com/" + objectName}, nil
	})

	req := newMultipartRequest(t, "/v1/app-package", map[string]string{
		"app_name": "demo-app",
		"version":  "1.2.3",
	}, "file", "demo.apk", []byte("fake apk contents"))
	req.Header.Set("Token", "test-token")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(capturedObjectNames) != 2 {
		t.Fatalf("upload calls = %d, want 2", len(capturedObjectNames))
	}

	historicalObjectName := capturedObjectNames[0]
	latestObjectName := capturedObjectNames[1]
	if !strings.HasPrefix(historicalObjectName, "apps/demo-app/1.2.3/") {
		t.Fatalf("object key %q does not include expected app/version prefix", historicalObjectName)
	}
	if latestObjectName != "apps/demo-app/latest/app.apk" {
		t.Fatalf("latest object key = %q, want stable APK alias", latestObjectName)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["nightly_download_url"]; ok {
		t.Fatal("nightly_download_url was returned for non-nightly upload")
	}
	if _, ok := body["nightly_object_key"]; ok {
		t.Fatal("nightly_object_key was returned for non-nightly upload")
	}
	if body["latest_object_key"] != latestObjectName {
		t.Fatalf("latest_object_key = %q, want %q", body["latest_object_key"], latestObjectName)
	}
	if body["latest_download_url"] != "https://cdn.example.com/"+latestObjectName {
		t.Fatalf("latest_download_url = %q, want CDN URL plus latest object key", body["latest_download_url"])
	}
}

func TestUploadAppPackageNonNightlyAABUsesLatestAlias(t *testing.T) {
	router := newAppPackageTestRouter(t)

	var capturedObjectNames []string
	stubUploadObject(t, func(ctx context.Context, objectName string, filePath string, objectSize int64) (*object.UploadResult, error) {
		capturedObjectNames = append(capturedObjectNames, objectName)
		return &object.UploadResult{URL: "https://cdn.example.com/" + objectName}, nil
	})

	req := newMultipartRequest(t, "/v1/app-package", map[string]string{
		"app_name": "demo-app",
		"version":  "2.0.0",
	}, "file", "demo.aab", []byte("fake aab contents"))
	req.Header.Set("Token", "test-token")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(capturedObjectNames) != 2 {
		t.Fatalf("upload calls = %d, want 2", len(capturedObjectNames))
	}
	if capturedObjectNames[1] != "apps/demo-app/latest/app.aab" {
		t.Fatalf("latest object key = %q, want stable AAB alias", capturedObjectNames[1])
	}
}

func TestUploadAppPackageValidationFailures(t *testing.T) {
	tests := []struct {
		name        string
		fields      map[string]string
		fileField   string
		filename    string
		wantMessage string
	}{
		{
			name:        "missing file",
			fields:      map[string]string{"app_name": "demo-app"},
			wantMessage: "file is required",
		},
		{
			name:        "blank app name",
			fields:      map[string]string{"app_name": "   "},
			fileField:   "file",
			filename:    "demo.apk",
			wantMessage: "app_name is required",
		},
		{
			name:        "unsupported extension",
			fields:      map[string]string{"app_name": "demo-app"},
			fileField:   "file",
			filename:    "demo.zip",
			wantMessage: "file must be .apk or .aab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newAppPackageTestRouter(t)
			called := false
			stubUploadObject(t, func(ctx context.Context, objectName string, filePath string, objectSize int64) (*object.UploadResult, error) {
				called = true
				return nil, errors.New("upload should not be called")
			})

			req := newMultipartRequest(t, "/v1/app-package", tt.fields, tt.fileField, tt.filename, []byte("fake package"))
			req.Header.Set("Token", "test-token")

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if called {
				t.Fatal("upload function was called for invalid request")
			}

			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body["message"] != tt.wantMessage {
				t.Fatalf("message = %q, want %q", body["message"], tt.wantMessage)
			}
		})
	}
}

func TestUploadAppPackageRequiresToken(t *testing.T) {
	router := newAppPackageTestRouter(t)
	called := false
	stubUploadObject(t, func(ctx context.Context, objectName string, filePath string, objectSize int64) (*object.UploadResult, error) {
		called = true
		return nil, errors.New("upload should not be called")
	})

	req := newMultipartRequest(t, "/v1/app-package", map[string]string{
		"app_name": "demo-app",
	}, "file", "demo.apk", []byte("fake package"))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("upload function was called for unauthorized request")
	}
}

func newAppPackageTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	conf.Conf.AuthToken = "test-token"
	conf.Conf.CorsHeader = []string{"Token", "Content-Type"}

	router := gin.New()
	Router(router)
	return router
}

func stubUploadObject(t *testing.T, fn appPackageUploader) {
	t.Helper()
	original := uploadObject
	uploadObject = fn
	t.Cleanup(func() {
		uploadObject = original
	})
}

func newMultipartRequest(t *testing.T, target string, fields map[string]string, fileField string, filename string, fileContent []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field %s: %v", key, err)
		}
	}
	if fileField != "" {
		part, err := writer.CreateFormFile(fileField, filename)
		if err != nil {
			t.Fatalf("create file field: %v", err)
		}
		if _, err := part.Write(fileContent); err != nil {
			t.Fatalf("write file content: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
