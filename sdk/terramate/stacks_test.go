package terramate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// setupTestServer creates a test server and client for testing
func setupTestServer(t *testing.T, handler http.HandlerFunc) (*Client, func()) {
	t.Helper()
	ts := httptest.NewServer(handler)
	c, err := NewClientWithAPIKey("test-api-key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	return c, ts.Close
}

func TestStacksList_ParsesResponse(t *testing.T) {
	payload := `{
		"stacks": [
			{
				"stack_id": 123,
				"repository": "github.com/acme/infrastructure",
				"target": "production",
				"path": "/stacks/vpc",
				"default_branch": "main",
				"meta_id": "vpc-prod-01",
				"meta_name": "Production VPC",
				"meta_description": "Main production VPC",
				"meta_tags": ["production", "networking"],
				"status": "ok",
				"deployment_status": "ok",
				"drift_status": "ok",
				"draft": false,
				"is_archived": false,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-15T12:00:00Z",
				"seen_at": "2024-01-15T12:00:00Z"
			}
		],
		"paginated_result": {
			"page": 1,
			"per_page": 10,
			"total": 1
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/stacks/org-uuid-123"
		if r.URL.Path != expectedPath {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expectedPath)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	result, resp, err := client.Stacks.List(context.Background(), "org-uuid-123", nil)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}

	// Verify response object
	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if resp.HTTPResponse.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.HTTPResponse.StatusCode)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
		return
	}
	if len(result.Stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(result.Stacks))
	}

	stack := result.Stacks[0]
	if stack.StackID != 123 {
		t.Errorf("unexpected stack_id: got %d, want 123", stack.StackID)
	}
	if stack.Repository != "github.com/acme/infrastructure" {
		t.Errorf("unexpected repository: got %s", stack.Repository)
	}
	if stack.MetaName != "Production VPC" {
		t.Errorf("unexpected meta_name: got %s", stack.MetaName)
	}
	if stack.Status != "ok" {
		t.Errorf("unexpected status: got %s", stack.Status)
	}
	if len(stack.MetaTags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(stack.MetaTags))
	}
}

func TestStacksList_WithOptions(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		query := r.URL.Query()
		if query.Get("page") != "2" {
			t.Errorf("expected page=2, got %s", query.Get("page"))
		}
		if query.Get("per_page") != "20" {
			t.Errorf("expected per_page=20, got %s", query.Get("per_page"))
		}
		if query.Get("status") != "ok,failed" {
			t.Errorf("expected status=ok,failed, got %s", query.Get("status"))
		}
		if query.Get("search") != "vpc" {
			t.Errorf("expected search=vpc, got %s", query.Get("search"))
		}

		payload := `{"stacks":[],"paginated_result":{"page":2,"per_page":20,"total":0}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	opts := &StacksListOptions{
		ListOptions: ListOptions{
			Page:    2,
			PerPage: 20,
		},
		Status: []string{"ok", "failed"},
		Search: "vpc",
	}

	_, _, err := client.Stacks.List(context.Background(), "org-uuid", opts)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
}

func TestStacksList_Validation(t *testing.T) {
	c, err := NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tests := []struct {
		name      string
		orgUUID   string
		wantError string
	}{
		{"empty org UUID", "", "organization UUID is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.Stacks.List(context.Background(), tt.orgUUID, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantError {
				t.Errorf("got error %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestStacksGet_ParsesResponse(t *testing.T) {
	payload := `{
		"stack_id": 456,
		"repository": "github.com/acme/infrastructure",
		"target": "staging",
		"path": "/stacks/database",
		"default_branch": "main",
		"meta_id": "db-staging-01",
		"meta_name": "Staging Database",
		"meta_description": "PostgreSQL database for staging",
		"meta_tags": ["staging", "database"],
		"status": "ok",
		"deployment_status": "ok",
		"drift_status": "ok",
		"draft": false,
		"is_archived": false,
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-15T12:00:00Z",
		"seen_at": "2024-01-15T12:00:00Z"
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/stacks/org-uuid-123/456"
		if r.URL.Path != expectedPath {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expectedPath)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	stack, resp, err := client.Stacks.Get(context.Background(), "org-uuid-123", 456)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

	// Verify response object
	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if resp.HTTPResponse.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.HTTPResponse.StatusCode)
	}

	if stack == nil {
		t.Fatal("expected non-nil stack")
		return
	}
	if stack.StackID != 456 {
		t.Errorf("unexpected stack_id: got %d, want 456", stack.StackID)
	}
	if stack.MetaName != "Staging Database" {
		t.Errorf("unexpected meta_name: got %s", stack.MetaName)
	}
	if stack.Target != "staging" {
		t.Errorf("unexpected target: got %s", stack.Target)
	}
}

func TestStacksGet_Validation(t *testing.T) {
	c, err := NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tests := []struct {
		name      string
		orgUUID   string
		stackID   int
		wantError string
	}{
		{"empty org UUID", "", 123, "organization UUID is required"},
		{"zero stack ID", "org-uuid", 0, "stack ID must be positive"},
		{"negative stack ID", "org-uuid", -1, "stack ID must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.Stacks.Get(context.Background(), tt.orgUUID, tt.stackID)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantError {
				t.Errorf("got error %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

//nolint:gocyclo // High complexity due to comprehensive field assertions
func TestStacksGet_ParsesAllFields(t *testing.T) {
	payload := `{
		"stack_id": 789,
		"repository": "github.com/acme/infrastructure",
		"target": "production",
		"path": "/stacks/vpc",
		"default_branch": "main",
		"meta_id": "vpc-prod-01",
		"meta_name": "Production VPC",
		"meta_description": "Main VPC for production",
		"meta_tags": ["production", "networking"],
		"status": "ok",
		"deployment_status": "ok",
		"drift_status": "ok",
		"draft": false,
		"is_archived": true,
		"archived_at": "2024-12-01T10:00:00Z",
		"archived_by_user_uuid": "123e4567-e89b-12d3-a456-426614174000",
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-15T12:00:00Z",
		"seen_at": "2024-01-15T12:00:00Z",
		"related_stacks": [
			{"stack_id": 790, "target": "staging"},
			{"stack_id": 791, "target": "development"}
		],
		"resources": {
			"count": 42,
			"policy_check": {
				"created_at": "2024-01-15T11:00:00Z",
				"passed": false,
				"counters": {
					"passed_count": 10,
					"severity_low_count": 2,
					"severity_medium_count": 1,
					"severity_high_count": 0
				}
			}
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	stack, resp, err := client.Stacks.Get(context.Background(), "org-uuid", 789)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

	// Verify response object
	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if resp.HTTPResponse.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.HTTPResponse.StatusCode)
	}

	// Test all fields
	if stack.StackID != 789 {
		t.Errorf("unexpected stack_id: got %d, want 789", stack.StackID)
	}
	if stack.IsArchived != true {
		t.Errorf("unexpected is_archived: got %v, want true", stack.IsArchived)
	}
	if stack.ArchivedAt == nil {
		t.Error("expected archived_at to be set")
	}
	if stack.ArchivedByUserUUID != "123e4567-e89b-12d3-a456-426614174000" {
		t.Errorf("unexpected archived_by_user_uuid: got %s", stack.ArchivedByUserUUID)
	}

	// Test related stacks
	if len(stack.RelatedStacks) != 2 {
		t.Fatalf("expected 2 related stacks, got %d", len(stack.RelatedStacks))
	}
	if stack.RelatedStacks[0].StackID != 790 {
		t.Errorf("unexpected related stack ID: got %d, want 790", stack.RelatedStacks[0].StackID)
	}
	if stack.RelatedStacks[0].Target != "staging" {
		t.Errorf("unexpected related stack target: got %s, want staging", stack.RelatedStacks[0].Target)
	}

	// Test resources
	if stack.Resources == nil {
		t.Fatal("expected resources to be set")
	}
	if stack.Resources.Count != 42 {
		t.Errorf("unexpected resource count: got %d, want 42", stack.Resources.Count)
	}

	// Test policy check
	if stack.Resources.PolicyCheck == nil {
		t.Fatal("expected policy_check to be set")
	}
	if stack.Resources.PolicyCheck.Passed != false {
		t.Error("expected policy check to have passed=false")
	}
	if stack.Resources.PolicyCheck.Counters.PassedCount != 10 {
		t.Errorf("unexpected passed_count: got %d, want 10", stack.Resources.PolicyCheck.Counters.PassedCount)
	}
	if stack.Resources.PolicyCheck.Counters.SeverityMediumCount != 1 {
		t.Errorf("unexpected severity_medium_count: got %d, want 1", stack.Resources.PolicyCheck.Counters.SeverityMediumCount)
	}
}

//nolint:gocyclo // High complexity due to comprehensive query parameter verification
func TestStacksList_WithAllQueryParameters(t *testing.T) {
	draft := true
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		// Verify all query parameters
		if query.Get("repository") != "github.com/acme/repo1,github.com/acme/repo2" {
			t.Errorf("unexpected repository: got %s", query.Get("repository"))
		}
		if query.Get("target") != "production,staging" {
			t.Errorf("unexpected target: got %s", query.Get("target"))
		}
		if query.Get("deployment_status") != "ok,failed" {
			t.Errorf("unexpected deployment_status: got %s", query.Get("deployment_status"))
		}
		if query.Get("drift_status") != "drifted" {
			t.Errorf("unexpected drift_status: got %s", query.Get("drift_status"))
		}
		if query.Get("draft") != "true" {
			t.Errorf("unexpected draft: got %s", query.Get("draft"))
		}
		if query.Get("is_archived") != "false,true" {
			t.Errorf("unexpected is_archived: got %s", query.Get("is_archived"))
		}
		if query.Get("meta_id") != "vpc-prod-01" {
			t.Errorf("unexpected meta_id: got %s", query.Get("meta_id"))
		}
		if query.Get("deployment_uuid") != "deploy-123" {
			t.Errorf("unexpected deployment_uuid: got %s", query.Get("deployment_uuid"))
		}

		// Verify meta_tag uses Add (multiple params)
		metaTags := query["meta_tag"]
		if len(metaTags) != 2 || metaTags[0] != "prod" || metaTags[1] != "network" {
			t.Errorf("unexpected meta_tag: got %v", metaTags)
		}

		if query.Get("policy_severity") != "high,medium" {
			t.Errorf("unexpected policy_severity: got %s", query.Get("policy_severity"))
		}

		// Verify sort uses Add (multiple params)
		sorts := query["sort"]
		if len(sorts) != 2 || sorts[0] != "name" || sorts[1] != "created_at" {
			t.Errorf("unexpected sort: got %v", sorts)
		}

		payload := `{"stacks":[],"paginated_result":{"page":1,"per_page":10,"total":0}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	opts := &StacksListOptions{
		Repository:       []string{"github.com/acme/repo1", "github.com/acme/repo2"},
		Target:           []string{"production", "staging"},
		DeploymentStatus: []string{"ok", "failed"},
		DriftStatus:      []string{"drifted"},
		Draft:            &draft,
		IsArchived:       []bool{false, true},
		MetaID:           "vpc-prod-01",
		DeploymentUUID:   "deploy-123",
		MetaTag:          []string{"prod", "network"},
		PolicySeverity:   []string{"high", "medium"},
		Sort:             []string{"name", "created_at"},
	}

	_, _, err := client.Stacks.List(context.Background(), "org-uuid", opts)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
}

// Test error responses
func TestStacksList_HandlesAPIError(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		if _, werr := w.Write([]byte(`{"error":"organization not found"}`)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.Stacks.List(context.Background(), "invalid-uuid", nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	// Check that error is an APIError
	if apiErr, ok := err.(*APIError); ok {
		if apiErr.StatusCode != 404 {
			t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
		}
		if apiErr.Message != "organization not found" {
			t.Errorf("unexpected error message: %s", apiErr.Message)
		}
	} else {
		t.Errorf("expected APIError type, got %T", err)
	}
}

func TestStacksGet_HandlesAPIError(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		if _, werr := w.Write([]byte(`{"error":"stack not found"}`)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.Stacks.Get(context.Background(), "org-uuid", 999)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	if apiErr, ok := err.(*APIError); ok {
		if apiErr.StatusCode != 404 {
			t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
		}
	} else {
		t.Errorf("expected APIError type, got %T", err)
	}
}

// Test authentication headers
func TestStacksList_SendsAuthHeader(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Fatal("expected basic auth header")
		}
		if username != "test-api-key" {
			t.Errorf("expected username 'test-api-key', got %s", username)
		}
		if password != "" {
			t.Errorf("expected empty password, got %s", password)
		}

		payload := `{"stacks":[],"paginated_result":{"page":1,"per_page":10,"total":0}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.Stacks.List(context.Background(), "org-uuid", nil)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
}

func TestStacksGet_SendsAuthHeader(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Fatal("expected basic auth header")
		}
		if username != "test-api-key" {
			t.Errorf("expected username 'test-api-key', got %s", username)
		}
		if password != "" {
			t.Errorf("expected empty password, got %s", password)
		}

		payload := `{
			"stack_id": 123,
			"repository": "github.com/acme/repo",
			"path": "/stacks/test",
			"default_branch": "main",
			"meta_id": "test",
			"status": "ok",
			"deployment_status": "ok",
			"drift_status": "ok",
			"draft": false,
			"is_archived": false,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
			"seen_at": "2024-01-01T00:00:00Z"
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.Stacks.Get(context.Background(), "org-uuid", 123)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
}

// Test context cancellation
func TestStacksList_RespectsContextCancellation(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Wait for context cancellation
		<-r.Context().Done()
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err := client.Stacks.List(ctx, "org-uuid", nil)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestStacksGet_RespectsContextCancellation(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Wait for context cancellation
		<-r.Context().Done()
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err := client.Stacks.Get(ctx, "org-uuid", 123)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

// Test context timeout
func TestStacksList_RespectsContextTimeout(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err := client.Stacks.List(ctx, "org-uuid", nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
