package mcpserver

import (
	"context"
	"encoding/base64"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/orvice/objr/internal/object"
	"github.com/orvice/objr/internal/upload"
)

func TestToolNames(t *testing.T) {
	want := []string{ToolPing, ToolUploadImage, ToolUploadAppPackage}
	if got := ToolNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ToolNames() = %v, want %v", got, want)
	}
}

func TestNewServerRegistersTools(t *testing.T) {
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := NewServer(newTestService(t, nil)).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect server: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	defer clientSession.Close()

	tools, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	got := map[string]bool{}
	for _, tool := range tools.Tools {
		got[tool.Name] = true
	}
	for _, name := range ToolNames() {
		if !got[name] {
			t.Fatalf("registered tools = %v, missing %q", got, name)
		}
	}
}

func TestHandlePing(t *testing.T) {
	_, output, err := handlePing(context.Background(), nil, PingInput{})
	if err != nil {
		t.Fatalf("handlePing() error = %v", err)
	}
	if output.Message != "pong" {
		t.Fatalf("message = %q, want pong", output.Message)
	}
}

func TestHandleUploadImage(t *testing.T) {
	var uploaded []string
	svc := newTestService(t, &uploaded)

	_, output, err := handleUploadImage(svc)(context.Background(), nil, UploadImageInput{
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("image")),
		Filename:      "demo.png",
	})
	if err != nil {
		t.Fatalf("handleUploadImage() error = %v", err)
	}

	if output.Message != "success" {
		t.Fatalf("message = %q, want success", output.Message)
	}
	if output.URL != "https://cdn.example.com/"+output.ObjectKey {
		t.Fatalf("url = %q, want CDN URL plus object key", output.URL)
	}
	if output.ObjectKey != "images/2026/4/18/11111111-2222-3333-4444-555555555555-demo.png" {
		t.Fatalf("object key = %q", output.ObjectKey)
	}
	if len(uploaded) != 1 || uploaded[0] != output.ObjectKey {
		t.Fatalf("uploaded objects = %v, want [%q]", uploaded, output.ObjectKey)
	}
}

func TestHandleUploadAppPackage(t *testing.T) {
	var uploaded []string
	svc := newTestService(t, &uploaded)

	_, output, err := handleUploadAppPackage(svc)(context.Background(), nil, UploadAppPackageInput{
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("aab")),
		Filename:      "demo.aab",
		AppName:       "demo-app",
		Version:       "1.2.3",
	})
	if err != nil {
		t.Fatalf("handleUploadAppPackage() error = %v", err)
	}

	if output.Version != "1.2.3" {
		t.Fatalf("version = %q, want 1.2.3", output.Version)
	}
	if output.LatestObjectKey != "apps/demo-app/latest/app.aab" {
		t.Fatalf("latest object key = %q", output.LatestObjectKey)
	}
	if output.LatestDownloadURL != "https://cdn.example.com/"+output.LatestObjectKey {
		t.Fatalf("latest download URL = %q", output.LatestDownloadURL)
	}
	if len(uploaded) != 2 {
		t.Fatalf("uploaded objects = %v, want 2 uploads", uploaded)
	}
}

func TestToolValidationErrors(t *testing.T) {
	svc := newTestService(t, nil)

	tests := []struct {
		name    string
		run     func() error
		wantErr string
	}{
		{
			name: "missing source",
			run: func() error {
				_, _, err := handleUploadImage(svc)(context.Background(), nil, UploadImageInput{})
				return err
			},
			wantErr: "file_path or content_base64 is required",
		},
		{
			name: "ambiguous source",
			run: func() error {
				_, _, err := handleUploadImage(svc)(context.Background(), nil, UploadImageInput{
					FilePath:      "demo.png",
					ContentBase64: "ZGVtbw==",
					Filename:      "demo.png",
				})
				return err
			},
			wantErr: "provide either file_path or content_base64, not both",
		},
		{
			name: "invalid base64",
			run: func() error {
				_, _, err := handleUploadImage(svc)(context.Background(), nil, UploadImageInput{
					ContentBase64: "%%%",
					Filename:      "demo.png",
				})
				return err
			},
			wantErr: "illegal base64",
		},
		{
			name: "blank app name",
			run: func() error {
				_, _, err := handleUploadAppPackage(svc)(context.Background(), nil, UploadAppPackageInput{
					ContentBase64: "ZGVtbw==",
					Filename:      "demo.apk",
				})
				return err
			},
			wantErr: "app_name is required",
		},
		{
			name: "unsupported package extension",
			run: func() error {
				_, _, err := handleUploadAppPackage(svc)(context.Background(), nil, UploadAppPackageInput{
					ContentBase64: "ZGVtbw==",
					Filename:      "demo.zip",
					AppName:       "demo-app",
				})
				return err
			},
			wantErr: "file must be .apk or .aab",
		},
		{
			name: "unreadable file",
			run: func() error {
				_, _, err := handleUploadImage(svc)(context.Background(), nil, UploadImageInput{
					FilePath: "/path/that/does/not/exist.png",
				})
				return err
			},
			wantErr: "no such file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func newTestService(t *testing.T, uploaded *[]string) *upload.Service {
	t.Helper()

	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	return &upload.Service{
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
			if uploaded != nil {
				*uploaded = append(*uploaded, objectName)
			}
			return &object.UploadResult{URL: "https://cdn.example.com/" + objectName}, nil
		},
	}
}
