package mcpserver

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/orvice/objr/internal/upload"
)

const (
	ToolPing             = "ping"
	ToolUploadImage      = "upload_image"
	ToolUploadAppPackage = "upload_app_package"
)

type PingInput struct{}

type PingOutput struct {
	Message string `json:"message" jsonschema:"service liveness message"`
}

type UploadImageInput struct {
	FilePath      string `json:"file_path,omitempty" jsonschema:"server-readable local image path"`
	ContentBase64 string `json:"content_base64,omitempty" jsonschema:"base64-encoded image content"`
	Filename      string `json:"filename,omitempty" jsonschema:"filename to use with content_base64 or as an override for file_path"`
}

type UploadAppPackageInput struct {
	FilePath      string `json:"file_path,omitempty" jsonschema:"server-readable local APK or AAB path"`
	ContentBase64 string `json:"content_base64,omitempty" jsonschema:"base64-encoded APK or AAB content"`
	Filename      string `json:"filename,omitempty" jsonschema:"filename to use with content_base64 or as an override for file_path"`
	AppName       string `json:"app_name" jsonschema:"application name used in object keys"`
	Version       string `json:"version,omitempty" jsonschema:"application version; defaults to nightly when omitted or blank"`
}

func ToolNames() []string {
	return []string{ToolPing, ToolUploadImage, ToolUploadAppPackage}
}

func NewServer(svc *upload.Service) *mcp.Server {
	if svc == nil {
		svc = upload.NewService()
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "objr",
		Title:   "objr",
		Version: "v0.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolPing,
		Description: "Check objr MCP server liveness.",
	}, handlePing)

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolUploadImage,
		Description: "Upload an image from file_path or content_base64 and return its CDN URL.",
	}, handleUploadImage(svc))

	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolUploadAppPackage,
		Description: "Upload an APK or AAB from file_path or content_base64 and return historical plus channel URLs.",
	}, handleUploadAppPackage(svc))

	return server
}

func NewHTTPHandler(svc *upload.Service) http.Handler {
	server := NewServer(svc)
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)
}

func handlePing(context.Context, *mcp.CallToolRequest, PingInput) (*mcp.CallToolResult, PingOutput, error) {
	return nil, PingOutput{Message: "pong"}, nil
}

func handleUploadImage(svc *upload.Service) mcp.ToolHandlerFor[UploadImageInput, upload.ImageResult] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input UploadImageInput) (*mcp.CallToolResult, upload.ImageResult, error) {
		result, err := svc.UploadImage(ctx, upload.Source{
			FilePath:      input.FilePath,
			ContentBase64: input.ContentBase64,
			Filename:      input.Filename,
		})
		if err != nil {
			return nil, upload.ImageResult{}, err
		}
		return nil, *result, nil
	}
}

func handleUploadAppPackage(svc *upload.Service) mcp.ToolHandlerFor[UploadAppPackageInput, upload.AppPackageResult] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input UploadAppPackageInput) (*mcp.CallToolResult, upload.AppPackageResult, error) {
		result, err := svc.UploadAppPackage(ctx, upload.AppPackageInput{
			Source: upload.Source{
				FilePath:      input.FilePath,
				ContentBase64: input.ContentBase64,
				Filename:      input.Filename,
			},
			AppName: input.AppName,
			Version: input.Version,
		})
		if err != nil {
			return nil, upload.AppPackageResult{}, err
		}
		return nil, *result, nil
	}
}
