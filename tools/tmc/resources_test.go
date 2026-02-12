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

func TestListResources_Success(t *testing.T) {
	payload := terramate.ResourcesListResponse{
		Resources: []terramate.Resource{
			{
				ResourceUUID: "f1c9ecfe-1a45-499b-ab6d-1aa0a8ea2f95",
				Descriptor: terramate.ResourceDescriptor{
					Address: "aws_vpc.main",
					Type:    "aws_vpc",
				},
				Status:  "ok",
				Drifted: false,
				Pending: false,
			},
		},
		PaginatedResult: terramate.PaginatedResult{
			Total:   1,
			Page:    1,
			PerPage: 20,
		},
	}
	body, _ := json.Marshal(payload)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/resources/org-uuid" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(body)
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListResources(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if result.IsError {
		text, _ := mcp.AsTextContent(result.Content[0])
		t.Fatalf("unexpected error: %s", text.Text)
	}

	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	var decoded terramate.ResourcesListResponse
	if err := json.Unmarshal([]byte(textContent.Text), &decoded); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(decoded.Resources) != 1 || decoded.Resources[0].ResourceUUID != "f1c9ecfe-1a45-499b-ab6d-1aa0a8ea2f95" {
		t.Errorf("unexpected decoded result: %+v", decoded)
	}
}

func TestListResources_StackIDFilter(t *testing.T) {
	var capturedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"resources":[],"paginated_result":{"total":0,"page":1,"per_page":20}}`))
	}))
	defer ts.Close()

	c, _ := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	tool := ListResources(c)
	_, _ = tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          42,
			},
		},
	})

	if capturedPath != "/v1/resources/org-uuid?stack_id=42" {
		t.Errorf("expected stack_id in query, got: %s", capturedPath)
	}
}

func TestGetResource_Success(t *testing.T) {
	payload := terramate.Resource{
		ResourceUUID: "res-uuid-123",
		Descriptor:   terramate.ResourceDescriptor{Address: "aws_vpc.main", Type: "aws_vpc"},
		Status:       "ok",
	}
	body, _ := json.Marshal(payload)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/resources/org-uuid/res-uuid-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(body)
	}))
	defer ts.Close()

	c, _ := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	tool := GetResource(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"resource_uuid":     "res-uuid-123",
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if result.IsError {
		text, _ := mcp.AsTextContent(result.Content[0])
		t.Fatalf("unexpected error: %s", text.Text)
	}

	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	var decoded terramate.Resource
	if err := json.Unmarshal([]byte(textContent.Text), &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.ResourceUUID != "res-uuid-123" {
		t.Errorf("unexpected resource: %+v", decoded)
	}
}
