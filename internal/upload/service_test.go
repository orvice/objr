package upload

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/orvice/objr/internal/object"
)

func TestNormalizeAppPackageVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "missing", version: "", want: DefaultAppPackageVersion},
		{name: "blank", version: " \t ", want: DefaultAppPackageVersion},
		{name: "provided", version: " 1.2.3 ", want: "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeAppPackageVersion(tt.version); got != tt.want {
				t.Fatalf("NormalizeAppPackageVersion() = %q, want %q", got, tt.want)
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
			if got := IsAllowedAppPackage(tt.filename); got != tt.want {
				t.Fatalf("IsAllowedAppPackage() = %v, want %v", got, tt.want)
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
			if got := SanitizePathSegment(tt.value); got != tt.want {
				t.Fatalf("SanitizePathSegment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestObjectNames(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	now := time.Date(2026, time.April, 18, 10, 30, 0, 0, time.UTC)

	if got, want := ImageObjectName("demo.png", now, id), "images/2026/4/18/11111111-2222-3333-4444-555555555555-demo.png"; got != want {
		t.Fatalf("ImageObjectName() = %q, want %q", got, want)
	}
	if got, want := AppPackageObjectName("demo-app", "1.2.3", "demo.apk", now, id), "apps/demo-app/1.2.3/2026/4/18/11111111-2222-3333-4444-555555555555-demo.apk"; got != want {
		t.Fatalf("AppPackageObjectName() = %q, want %q", got, want)
	}
	if got, want := AppPackageAliasObjectName("demo-app", "nightly", "demo.APK"), "apps/demo-app/nightly/app.apk"; got != want {
		t.Fatalf("AppPackageAliasObjectName() = %q, want %q", got, want)
	}
}

func TestUploadImageFromBase64(t *testing.T) {
	svc := newTestService(t)

	result, err := svc.UploadImage(context.Background(), Source{
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("image")),
		Filename:      "demo.png",
	})
	if err != nil {
		t.Fatalf("UploadImage() error = %v", err)
	}

	if result.Message != "success" {
		t.Fatalf("message = %q, want success", result.Message)
	}
	if result.URL != "https://cdn.example.com/"+result.ObjectKey {
		t.Fatalf("url = %q, want CDN URL plus object key", result.URL)
	}
	if !strings.HasPrefix(result.ObjectKey, "images/2026/4/18/11111111-2222-3333-4444-555555555555-demo.png") {
		t.Fatalf("object key = %q", result.ObjectKey)
	}
}

func TestUploadAppPackageFromBase64(t *testing.T) {
	svc := newTestService(t)

	result, err := svc.UploadAppPackage(context.Background(), AppPackageInput{
		Source: Source{
			ContentBase64: base64.StdEncoding.EncodeToString([]byte("apk")),
			Filename:      "demo.apk",
		},
		AppName: "demo-app",
	})
	if err != nil {
		t.Fatalf("UploadAppPackage() error = %v", err)
	}

	if result.Version != DefaultAppPackageVersion {
		t.Fatalf("version = %q, want %q", result.Version, DefaultAppPackageVersion)
	}
	if result.NightlyObjectKey != "apps/demo-app/nightly/app.apk" {
		t.Fatalf("nightly object key = %q", result.NightlyObjectKey)
	}
}

func TestResolveSourceValidation(t *testing.T) {
	tests := []struct {
		name   string
		source Source
	}{
		{name: "missing", source: Source{}},
		{name: "ambiguous", source: Source{FilePath: "demo.png", ContentBase64: "ZGVtbw=="}},
		{name: "base64 missing filename", source: Source{ContentBase64: "ZGVtbw=="}},
		{name: "invalid base64", source: Source{ContentBase64: "%%%", Filename: "demo.png"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolveSource(tt.source)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	return &Service{
		Now: func() time.Time {
			return time.Date(2026, time.April, 18, 10, 30, 0, 0, time.UTC)
		},
		NewUUID: func() (uuid.UUID, error) {
			return id, nil
		},
		Uploader: func(ctx context.Context, objectName string, filePath string, objectSize int64) (*object.UploadResult, error) {
			if _, err := os.Stat(filePath); err != nil {
				t.Fatalf("expected upload file to exist: %v", err)
			}
			if objectSize == 0 {
				t.Fatal("expected non-zero object size")
			}
			return &object.UploadResult{URL: "https://cdn.example.com/" + objectName}, nil
		},
	}
}
