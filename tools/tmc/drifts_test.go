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

func TestListDrifts_Success(t *testing.T) {
	payload := `{
		"drifts": [
			{
				"id": 100,
				"org_uuid": "org-uuid-123",
				"stack_id": 456,
				"status": "drifted",
				"metadata": {"key": "value"},
				"started_at": "2024-01-15T10:00:00Z",
				"finished_at": "2024-01-15T10:05:00Z",
				"auth_type": "gha",
				"grouping_key": "repo+id+1",
				"cmd": ["terraform", "plan"]
			}
		],
		"paginated_result": {
			"page": 1,
			"per_page": 10,
			"total": 1
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/stacks/org-uuid/456/drifts" {
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

	tool := ListDrifts(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(456),
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
	var response terramate.DriftsListResponse
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(response.Drifts) != 1 {
		t.Fatalf("expected 1 drift, got %d", len(response.Drifts))
	}
	if response.Drifts[0].ID != 100 {
		t.Fatalf("expected id=100, got %d", response.Drifts[0].ID)
	}
	if response.PaginatedResult.Total != 1 {
		t.Fatalf("expected total=1, got %d", response.PaginatedResult.Total)
	}
}

func TestListDrifts_WithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("drift_status") != "drifted,failed" {
			t.Errorf("expected drift_status=drifted,failed, got %s", query.Get("drift_status"))
		}
		if query.Get("grouping_key") != "repo+id+1" {
			t.Errorf("expected grouping_key=repo+id+1, got %s", query.Get("grouping_key"))
		}
		if query.Get("page") != "2" {
			t.Errorf("expected page=2, got %s", query.Get("page"))
		}
		if query.Get("per_page") != "10" {
			t.Errorf("expected per_page=10, got %s", query.Get("per_page"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(`{"drifts":[],"paginated_result":{"total":0,"page":2,"per_page":10}}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListDrifts(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(123),
				"drift_status":      []interface{}{"drifted", "failed"},
				"grouping_key":      "repo+id+1",
				"page":              float64(2),
				"per_page":          float64(10),
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

func TestListDrifts_MissingOrgUUID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListDrifts(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"stack_id": float64(123),
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

func TestListDrifts_MissingStackID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListDrifts(c)
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
		t.Fatal("expected error result for missing stack_id")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Stack ID is required and must be a number." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestListDrifts_InvalidStackID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListDrifts(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(0),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid stack_id")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Stack ID must be positive." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestListDrifts_InvalidPerPage(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListDrifts(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(123),
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
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Per page value must not exceed 100." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestListDrifts_Unauthorized(t *testing.T) {
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

	tool := ListDrifts(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(123),
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

func TestListDrifts_NotFound(t *testing.T) {
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

	tool := ListDrifts(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(999),
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
	if textContent.Text != "Stack with ID 999 not found." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestGetDrift_Success(t *testing.T) {
	payload := `{
		"id": 100,
		"org_uuid": "org-uuid-123",
		"stack_id": 456,
		"status": "drifted",
		"metadata": {"branch": "main"},
		"started_at": "2024-01-15T10:00:00Z",
		"finished_at": "2024-01-15T10:05:00Z",
		"auth_type": "gha",
		"grouping_key": "repo+id+1",
		"cmd": ["terraform", "plan"],
		"drift_details": {
			"provisioner": "terraform",
			"serial": 42,
			"changeset_ascii": "Terraform will perform the following actions:\n\n  + resource.new\n",
			"changeset_json": "{\"resource_changes\":[]}"
		},
		"stack": {
			"stack_id": 456,
			"repository": "github.com/acme/infra",
			"path": "/stacks/vpc",
			"default_branch": "main",
			"meta_id": "vpc",
			"status": "ok",
			"deployment_status": "ok",
			"drift_status": "drifted",
			"draft": false,
			"is_archived": false,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-02T00:00:00Z"
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/drifts/org-uuid/456/100" {
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

	tool := GetDrift(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(456),
				"drift_id":          float64(100),
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
	var drift terramate.Drift
	if err := json.Unmarshal([]byte(textContent.Text), &drift); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if drift.ID != 100 {
		t.Fatalf("expected id=100, got %d", drift.ID)
	}
	if drift.DriftDetails == nil {
		t.Fatal("expected drift_details to be set")
	}
	if drift.DriftDetails.ChangesetASCII != "Terraform will perform the following actions:\n\n  + resource.new\n" {
		t.Fatalf("unexpected changeset_ascii: %s", drift.DriftDetails.ChangesetASCII)
	}
	if drift.Stack == nil {
		t.Fatal("expected stack to be set")
	}
	if drift.Stack.StackID != 456 {
		t.Fatalf("expected stack.stack_id=456, got %d", drift.Stack.StackID)
	}
}

func TestGetDrift_MissingOrgUUID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetDrift(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"stack_id": float64(456),
				"drift_id": float64(100),
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

func TestGetDrift_MissingStackID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetDrift(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"drift_id":          float64(100),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for missing stack_id")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Stack ID is required and must be a number." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestGetDrift_MissingDriftID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetDrift(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(456),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for missing drift_id")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Drift ID is required and must be a number." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestGetDrift_InvalidStackID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetDrift(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(0),
				"drift_id":          float64(100),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid stack_id")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Stack ID must be positive." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestGetDrift_InvalidDriftID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetDrift(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(456),
				"drift_id":          float64(0),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid drift_id")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Drift ID must be positive." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestGetDrift_Unauthorized(t *testing.T) {
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

	tool := GetDrift(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(456),
				"drift_id":          float64(100),
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

func TestGetDrift_NotFound(t *testing.T) {
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

	tool := GetDrift(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(456),
				"drift_id":          float64(999),
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
	if textContent.Text != "Drift with ID 999 not found for stack 456." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}
