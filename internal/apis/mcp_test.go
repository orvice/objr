package apis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPRequiresToken(t *testing.T) {
	router := newAppPackageTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMCPStreamableEndpointListsToolsWithToken(t *testing.T) {
	router := newAppPackageTestRouter(t)
	server := httptest.NewServer(router)
	defer server.Close()

	httpClient := &http.Client{
		Transport: tokenRoundTripper{
			token: "test-token",
			base:  http.DefaultTransport,
		},
	}
	transport := &mcp.StreamableClientTransport{
		Endpoint:             server.URL + "/mcp",
		HTTPClient:           httpClient,
		DisableStandaloneSSE: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "objr-test", Version: "v0.0.0"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list MCP tools: %v", err)
	}

	got := map[string]bool{}
	for _, tool := range tools.Tools {
		got[tool.Name] = true
	}
	for _, name := range []string{"ping", "upload_image", "upload_app_package"} {
		if !got[name] {
			t.Fatalf("tools = %v, missing %q", got, name)
		}
	}
}

type tokenRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (t tokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	clone := req.Clone(req.Context())
	clone.Header.Set("Token", t.token)
	return base.RoundTrip(clone)
}
