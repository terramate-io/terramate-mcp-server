package terramate

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestReviewRequestsList_ParsesResponse(t *testing.T) {
	payload := `{
		"review_requests": [
			{
				"review_request_id": 42,
				"platform": "github",
				"repository": "github.com/acme/infrastructure",
				"commit_sha": "abc123def456",
				"number": 123,
				"title": "feat: Add VPC configuration",
				"description": "This PR adds VPC config",
				"url": "https://github.com/acme/infrastructure/pull/123",
				"branch": "feature/vpc",
				"base_branch": "main",
				"status": "open",
				"draft": false,
				"review_decision": "review_required",
				"approved_count": 0,
				"changes_requested_count": 0,
				"checks_total_count": 5,
				"checks_success_count": 5,
				"checks_failure_count": 0,
				"platform_created_at": "2024-01-15T10:00:00Z",
				"platform_updated_at": "2024-01-15T12:00:00Z"
			}
		],
		"paginated_result": {
			"page": 1,
			"per_page": 10,
			"total": 1
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/review_requests/org-uuid-123"
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

	result, resp, err := client.ReviewRequests.List(context.Background(), "org-uuid-123", nil)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}

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
	if len(result.ReviewRequests) != 1 {
		t.Fatalf("expected 1 review request, got %d", len(result.ReviewRequests))
	}

	rr := result.ReviewRequests[0]
	if rr.ReviewRequestID != 42 {
		t.Errorf("unexpected review_request_id: got %d, want 42", rr.ReviewRequestID)
	}
	if rr.Number != 123 {
		t.Errorf("unexpected number: got %d, want 123", rr.Number)
	}
	if rr.Title != "feat: Add VPC configuration" {
		t.Errorf("unexpected title: got %s", rr.Title)
	}
	if rr.Status != "open" {
		t.Errorf("unexpected status: got %s", rr.Status)
	}
}

func TestReviewRequestsList_WithOptions(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("page") != "2" {
			t.Errorf("expected page=2, got %s", query.Get("page"))
		}
		if query.Get("per_page") != "20" {
			t.Errorf("expected per_page=20, got %s", query.Get("per_page"))
		}
		if query.Get("status") != "open,merged" {
			t.Errorf("expected status=open,merged, got %s", query.Get("status"))
		}
		if query.Get("repository") != "github.com/acme/repo" {
			t.Errorf("expected repository=github.com/acme/repo, got %s", query.Get("repository"))
		}
		if query.Get("search") != "vpc" {
			t.Errorf("expected search=vpc, got %s", query.Get("search"))
		}

		payload := `{"review_requests":[],"paginated_result":{"page":2,"per_page":20,"total":0}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	opts := &ReviewRequestsListOptions{
		ListOptions: ListOptions{
			Page:    2,
			PerPage: 20,
		},
		Status:     []string{"open", "merged"},
		Repository: []string{"github.com/acme/repo"},
		Search:     "vpc",
	}

	_, _, err := client.ReviewRequests.List(context.Background(), "org-uuid", opts)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
}

func TestReviewRequestsList_Validation(t *testing.T) {
	c, err := NewClient("key")
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
			_, _, err := c.ReviewRequests.List(context.Background(), tt.orgUUID, nil)
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
func TestReviewRequestsGet_ParsesResponse(t *testing.T) {
	payload := `{
		"review_request": {
			"review_request_id": 42,
			"platform": "github",
			"repository": "github.com/acme/infrastructure",
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
					"repository": "github.com/acme/infrastructure",
					"path": "/stacks/vpc",
					"default_branch": "main",
					"meta_id": "vpc-prod",
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
					"serial": 42,
					"changeset_ascii": "Terraform will perform the following actions..."
				}
			}
		]
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/review_requests/org-uuid/42"
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

	result, resp, err := client.ReviewRequests.Get(context.Background(), "org-uuid", 42, nil)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

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
	if result.ReviewRequest.ReviewRequestID != 42 {
		t.Errorf("unexpected review_request_id: got %d, want 42", result.ReviewRequest.ReviewRequestID)
	}
	if result.ReviewRequest.Number != 123 {
		t.Errorf("unexpected number: got %d, want 123", result.ReviewRequest.Number)
	}
	if result.ReviewRequest.Title != "feat: Add VPC" {
		t.Errorf("unexpected title: got %s", result.ReviewRequest.Title)
	}

	if len(result.StackPreviews) != 1 {
		t.Fatalf("expected 1 stack preview, got %d", len(result.StackPreviews))
	}

	sp := result.StackPreviews[0]
	if sp.StackPreviewID != 100 {
		t.Errorf("unexpected stack_preview_id: got %d, want 100", sp.StackPreviewID)
	}
	if sp.Status != "changed" {
		t.Errorf("unexpected status: got %s", sp.Status)
	}
	if sp.Stack == nil {
		t.Fatal("expected stack to be set")
	}
	if sp.Stack.StackID != 456 {
		t.Errorf("unexpected stack.stack_id: got %d, want 456", sp.Stack.StackID)
	}
	if sp.ChangesetDetails == nil {
		t.Fatal("expected changeset_details to be set")
	}
}

func TestReviewRequestsGet_ExcludeStackPreviews(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("exclude_stack_previews") != "true" {
			t.Errorf("expected exclude_stack_previews=true, got %s", query.Get("exclude_stack_previews"))
		}

		payload := `{
			"review_request": {
				"review_request_id": 42,
				"number": 123,
				"title": "Test PR",
				"status": "open"
			},
			"stack_previews": null
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	opts := &ReviewRequestGetOptions{
		ExcludeStackPreviews: true,
	}

	_, _, err := client.ReviewRequests.Get(context.Background(), "org-uuid", 42, opts)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
}

func TestReviewRequestsGet_Validation(t *testing.T) {
	c, err := NewClient("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tests := []struct {
		name            string
		orgUUID         string
		reviewRequestID int
		wantError       string
	}{
		{"empty org UUID", "", 42, "organization UUID is required"},
		{"zero review request ID", "org-uuid", 0, "review request ID must be positive"},
		{"negative review request ID", "org-uuid", -1, "review request ID must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.ReviewRequests.Get(context.Background(), tt.orgUUID, tt.reviewRequestID, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantError {
				t.Errorf("got error %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestReviewRequestsList_HandlesAPIError(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		if _, werr := w.Write([]byte(`{"error":"organization not found"}`)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.ReviewRequests.List(context.Background(), "invalid-uuid", nil)
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

func TestReviewRequestsGet_HandlesAPIError(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		if _, werr := w.Write([]byte(`{"error":"review request not found"}`)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.ReviewRequests.Get(context.Background(), "org-uuid", 999, nil)
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

func TestReviewRequestsList_SendsAuthHeader(t *testing.T) {
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

		payload := `{"review_requests":[],"paginated_result":{"page":1,"per_page":10,"total":0}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.ReviewRequests.List(context.Background(), "org-uuid", nil)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
}

func TestReviewRequestsGet_SendsAuthHeader(t *testing.T) {
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
			"review_request": {"review_request_id": 42, "number": 123, "title": "Test", "status": "open"}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.ReviewRequests.Get(context.Background(), "org-uuid", 42, nil)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
}

func TestReviewRequestsList_RespectsContextCancellation(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := client.ReviewRequests.List(ctx, "org-uuid", nil)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestReviewRequestsGet_RespectsContextCancellation(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := client.ReviewRequests.Get(ctx, "org-uuid", 42, nil)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestReviewRequestsList_RespectsContextTimeout(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err := client.ReviewRequests.List(ctx, "org-uuid", nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
