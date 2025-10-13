package terramate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/stacks/org-uuid-123"
		if r.URL.Path != expectedPath {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expectedPath)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	}))
	defer ts.Close()

	c, err := NewClient("key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	result, _, err := c.Stacks.List(context.Background(), "org-uuid-123", nil)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer ts.Close()

	c, err := NewClient("key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	opts := &StacksListOptions{
		ListOptions: ListOptions{
			Page:    2,
			PerPage: 20,
		},
		Status: []string{"ok", "failed"},
		Search: "vpc",
	}

	_, _, err = c.Stacks.List(context.Background(), "org-uuid", opts)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
}

func TestStacksList_RequiresOrgUUID(t *testing.T) {
	c, err := NewClient("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, _, err = c.Stacks.List(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error when org_uuid is empty")
	}
	if err.Error() != "organization UUID is required" {
		t.Errorf("unexpected error message: %v", err)
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

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/stacks/org-uuid-123/456"
		if r.URL.Path != expectedPath {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expectedPath)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	}))
	defer ts.Close()

	c, err := NewClient("key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	stack, _, err := c.Stacks.Get(context.Background(), "org-uuid-123", 456)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

	if stack == nil {
		t.Fatal("expected non-nil stack")
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

func TestStacksGet_RequiresOrgUUID(t *testing.T) {
	c, err := NewClient("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, _, err = c.Stacks.Get(context.Background(), "", 123)
	if err == nil {
		t.Fatal("expected error when org_uuid is empty")
	}
	if err.Error() != "organization UUID is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestStacksGet_RequiresPositiveStackID(t *testing.T) {
	c, err := NewClient("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, _, err = c.Stacks.Get(context.Background(), "org-uuid", 0)
	if err == nil {
		t.Fatal("expected error when stack_id is 0")
	}
	if err.Error() != "stack ID must be positive" {
		t.Errorf("unexpected error message: %v", err)
	}

	_, _, err = c.Stacks.Get(context.Background(), "org-uuid", -1)
	if err == nil {
		t.Fatal("expected error when stack_id is negative")
	}
}

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

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	}))
	defer ts.Close()

	c, err := NewClient("key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	stack, _, err := c.Stacks.Get(context.Background(), "org-uuid", 789)
	if err != nil {
		t.Fatalf("Get error: %v", err)
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
