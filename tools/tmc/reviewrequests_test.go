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

func TestListReviewRequests_Success(t *testing.T) {
	payload := `{
		"review_requests": [
			{
				"review_request_id": 42,
				"platform": "github",
				"repository": "github.com/acme/infra",
				"number": 123,
				"title": "feat: Add VPC",
				"status": "open",
				"branch": "feature/vpc",
				"base_branch": "main",
				"preview": {
					"id": 100,
					"status": "current",
					"affected_count": 0,
					"pending_count": 0,
					"running_count": 0,
					"changed_count": 2,
					"unchanged_count": 1,
					"failed_count": 0,
					"canceled_count": 0
				}
			}
		],
		"paginated_result": {
			"total": 1,
			"page": 1,
			"per_page": 10
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/review_requests/org-uuid" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(payload)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListReviewRequests(c)
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
	var response terramate.ReviewRequestsListResponse
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(response.ReviewRequests) != 1 {
		t.Fatalf("expected 1 review request, got %d", len(response.ReviewRequests))
	}
	if response.ReviewRequests[0].ReviewRequestID != 42 {
		t.Fatalf("expected review_request_id=42, got %d", response.ReviewRequests[0].ReviewRequestID)
	}
	if response.PaginatedResult.Total != 1 {
		t.Fatalf("expected total=1, got %d", response.PaginatedResult.Total)
	}
}

func TestListReviewRequests_WithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("status") != "open,merged" {
			t.Errorf("expected status=open,merged, got %s", query.Get("status"))
		}
		if query.Get("repository") != "github.com/acme/repo" {
			t.Errorf("expected repository=github.com/acme/repo, got %s", query.Get("repository"))
		}
		if query.Get("search") != "vpc" {
			t.Errorf("expected search=vpc, got %s", query.Get("search"))
		}
		if query.Get("page") != "2" {
			t.Errorf("expected page=2, got %s", query.Get("page"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(`{"review_requests":[],"paginated_result":{"total":0,"page":2,"per_page":10}}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListReviewRequests(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"status":            []interface{}{"open", "merged"},
				"repository":        []interface{}{"github.com/acme/repo"},
				"search":            "vpc",
				"page":              float64(2),
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

func TestListReviewRequests_MissingOrgUUID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListReviewRequests(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for missing org_uuid")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Organization UUID is required and must be a string." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestListReviewRequests_InvalidPerPage(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListReviewRequests(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"per_page":          float64(150),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for per_page > 100")
	}
}

func TestListReviewRequests_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		if _, err := w.Write([]byte(`{"error":"unauthorized"}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListReviewRequests(c)
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

func TestGetReviewRequest_Success(t *testing.T) {
	payload := `{
		"review_request": {
			"review_request_id": 42,
			"platform": "github",
			"repository": "github.com/acme/infra",
			"number": 123,
			"title": "feat: Add VPC",
			"status": "open",
			"branch": "feature/vpc",
			"base_branch": "main"
		},
		"stack_previews": [
			{
				"stack_preview_id": 100,
				"status": "changed",
				"technology": "terraform",
				"updated_at": "2024-01-15T12:00:00Z",
				"stack": {
					"stack_id": 456,
					"repository": "github.com/acme/infra",
					"path": "/stacks/vpc",
					"default_branch": "main",
					"meta_id": "vpc",
					"status": "ok",
					"deployment_status": "ok",
					"drift_status": "ok",
					"draft": false,
					"is_archived": false,
					"created_at": "2024-01-01T00:00:00Z",
					"updated_at": "2024-01-15T12:00:00Z",
					"seen_at": "2024-01-15T12:00:00Z"
				},
				"changeset_details": {
					"provisioner": "terraform",
					"changeset_ascii": "Terraform plan output..."
				}
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/review_requests/org-uuid/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(payload)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetReviewRequest(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"review_request_id": float64(42),
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
	var response terramate.ReviewRequestGetResponse
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response.ReviewRequest.ReviewRequestID != 42 {
		t.Fatalf("expected review_request_id=42, got %d", response.ReviewRequest.ReviewRequestID)
	}
	if len(response.StackPreviews) != 1 {
		t.Fatalf("expected 1 stack preview, got %d", len(response.StackPreviews))
	}
	if response.StackPreviews[0].Stack.StackID != 456 {
		t.Fatalf("expected stack_id=456, got %d", response.StackPreviews[0].Stack.StackID)
	}
}

func TestGetReviewRequest_MissingOrgUUID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetReviewRequest(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"review_request_id": float64(42),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for missing org_uuid")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Organization UUID is required and must be a string." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestGetReviewRequest_MissingReviewRequestID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetReviewRequest(c)
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
	if !result.IsError {
		t.Fatal("expected error result for missing review_request_id")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Review Request ID is required and must be a number." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestGetReviewRequest_InvalidReviewRequestID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetReviewRequest(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"review_request_id": float64(0),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid review_request_id")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Review Request ID must be positive." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestGetReviewRequest_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		if _, err := w.Write([]byte(`{"error":"unauthorized"}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetReviewRequest(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"review_request_id": float64(42),
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

func TestGetReviewRequest_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		if _, err := w.Write([]byte(`{"error":"not found"}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetReviewRequest(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"review_request_id": float64(999),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for 404")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Review Request with ID 999 not found." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}
