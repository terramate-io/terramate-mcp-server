package tmc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

func TestGetStackPreviewLogs_Success(t *testing.T) {
	payload := `{
		"stack_preview_log_lines": [
			{
				"log_line": 1,
				"timestamp": "2024-01-15T10:00:00Z",
				"channel": "stderr",
				"message": "Error: Provider authentication failed"
			},
			{
				"log_line": 2,
				"timestamp": "2024-01-15T10:00:01Z",
				"channel": "stderr",
				"message": "AWS_ACCESS_KEY_ID environment variable not set"
			}
		],
		"paginated_result": {
			"total": 2,
			"page": 1,
			"per_page": 100
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/stack_previews/org-uuid/100/logs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(payload)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClient("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStackPreviewLogs(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_preview_id":  float64(100),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if result.IsError {
		textContent, ok := mcp.AsTextContent(result.Content[0])
		if !ok {
			t.Fatal("expected TextContent")
		}
		t.Fatalf("unexpected error result: %v", textContent.Text)
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	var response terramate.StackPreviewLogsResponse
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(response.StackPreviewLogLines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(response.StackPreviewLogLines))
	}
	if response.StackPreviewLogLines[0].Channel != "stderr" {
		t.Errorf("unexpected channel: got %s", response.StackPreviewLogLines[0].Channel)
	}
}

func TestGetStackPreviewLogs_WithChannel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("channel") != "stderr" {
			t.Errorf("expected channel=stderr, got %s", query.Get("channel"))
		}

		payload := `{"stack_preview_log_lines":[],"paginated_result":{"total":0,"page":1,"per_page":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(payload)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClient("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStackPreviewLogs(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_preview_id":  float64(100),
				"channel":           "stderr",
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if result.IsError {
		textContent, ok := mcp.AsTextContent(result.Content[0])
		if !ok {
			t.Fatal("expected TextContent")
		}
		t.Fatalf("unexpected error result: %v", textContent.Text)
	}
}

func TestGetStackPreviewLogs_MissingOrgUUID(t *testing.T) {
	c, err := terramate.NewClient("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStackPreviewLogs(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"stack_preview_id": float64(100),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for missing org_uuid")
	}
}

func TestGetStackPreviewLogs_InvalidPreviewID(t *testing.T) {
	c, err := terramate.NewClient("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStackPreviewLogs(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_preview_id":  float64(0),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid preview_id")
	}
}

func TestGetStackPreviewLogs_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		if _, err := w.Write([]byte(`{"error":"unauthorized"}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClient("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStackPreviewLogs(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_preview_id":  float64(100),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for 401")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != terramate.ErrAuthenticationFailed {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestGetStackPreviewLogs_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		if _, err := w.Write([]byte(`{"error":"not found"}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClient("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStackPreviewLogs(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_preview_id":  float64(999),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for 404")
	}
}
