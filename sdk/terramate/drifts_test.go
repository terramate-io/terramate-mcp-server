package terramate

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestDriftsListForStack_ParsesResponse(t *testing.T) {
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

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/stacks/org-uuid-123/456/drifts"
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

	result, resp, err := client.Drifts.ListForStack(context.Background(), "org-uuid-123", 456, nil)
	if err != nil {
		t.Fatalf("ListForStack error: %v", err)
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
	if len(result.Drifts) != 1 {
		t.Fatalf("expected 1 drift, got %d", len(result.Drifts))
	}

	drift := result.Drifts[0]
	if drift.ID != 100 {
		t.Errorf("unexpected id: got %d, want 100", drift.ID)
	}
	if drift.OrgUUID != "org-uuid-123" {
		t.Errorf("unexpected org_uuid: got %s", drift.OrgUUID)
	}
	if drift.StackID != 456 {
		t.Errorf("unexpected stack_id: got %d, want 456", drift.StackID)
	}
	if drift.Status != "drifted" {
		t.Errorf("unexpected status: got %s", drift.Status)
	}
	if drift.AuthType != "gha" {
		t.Errorf("unexpected auth_type: got %s", drift.AuthType)
	}
	if drift.GroupingKey != "repo+id+1" {
		t.Errorf("unexpected grouping_key: got %s", drift.GroupingKey)
	}
	if len(drift.Cmd) != 2 {
		t.Errorf("expected 2 cmd elements, got %d", len(drift.Cmd))
	}
}

func TestDriftsListForStack_WithOptions(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		query := r.URL.Query()
		if query.Get("page") != "2" {
			t.Errorf("expected page=2, got %s", query.Get("page"))
		}
		if query.Get("per_page") != "20" {
			t.Errorf("expected per_page=20, got %s", query.Get("per_page"))
		}
		if query.Get("drift_status") != "drifted,failed" {
			t.Errorf("expected drift_status=drifted,failed, got %s", query.Get("drift_status"))
		}
		if query.Get("grouping_key") != "repo+id+1" {
			t.Errorf("expected grouping_key=repo+id+1, got %s", query.Get("grouping_key"))
		}

		payload := `{"drifts":[],"paginated_result":{"page":2,"per_page":20,"total":0}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	opts := &DriftsListOptions{
		ListOptions: ListOptions{
			Page:    2,
			PerPage: 20,
		},
		DriftStatus: []string{"drifted", "failed"},
		GroupingKey: "repo+id+1",
	}

	_, _, err := client.Drifts.ListForStack(context.Background(), "org-uuid", 123, opts)
	if err != nil {
		t.Fatalf("ListForStack error: %v", err)
	}
}

func TestDriftsListForStack_Validation(t *testing.T) {
	c, err := NewClient("key")
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
			_, _, err := c.Drifts.ListForStack(context.Background(), tt.orgUUID, tt.stackID, nil)
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
func TestDriftsListForStack_ParsesAllFields(t *testing.T) {
	payload := `{
		"drifts": [
			{
				"id": 200,
				"org_uuid": "org-uuid-456",
				"stack_id": 789,
				"status": "ok",
				"metadata": {"branch": "main", "commit": "abc123"},
				"started_at": "2024-01-20T14:00:00Z",
				"finished_at": "2024-01-20T14:10:00Z",
				"auth_type": "idp",
				"auth_user": {
					"display_name": "John Doe",
					"position": "DevOps Engineer",
					"user_picture_url": "https://example.com/avatar.jpg"
				},
				"grouping_key": "repo+id+2",
				"cmd": ["tofu", "plan", "-out=plan.out"]
			},
			{
				"id": 201,
				"org_uuid": "org-uuid-456",
				"stack_id": 789,
				"status": "failed",
				"metadata": {"error": "timeout"},
				"started_at": "2024-01-21T09:00:00Z",
				"finished_at": "2024-01-21T09:05:00Z",
				"auth_type": "gitlabcicd",
				"auth_trust": {
					"auth_id": "project/repo/.gitlab-ci.yml@refs/heads/main"
				},
				"grouping_key": "repo+id+3",
				"cmd": ["terraform", "plan"]
			}
		],
		"paginated_result": {
			"page": 1,
			"per_page": 10,
			"total": 2
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

	result, resp, err := client.Drifts.ListForStack(context.Background(), "org-uuid-456", 789, nil)
	if err != nil {
		t.Fatalf("ListForStack error: %v", err)
	}

	// Verify response object
	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if resp.HTTPResponse.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.HTTPResponse.StatusCode)
	}

	if len(result.Drifts) != 2 {
		t.Fatalf("expected 2 drifts, got %d", len(result.Drifts))
	}

	// Test first drift with auth_user
	drift1 := result.Drifts[0]
	if drift1.ID != 200 {
		t.Errorf("unexpected id: got %d, want 200", drift1.ID)
	}
	if drift1.Status != "ok" {
		t.Errorf("unexpected status: got %s, want ok", drift1.Status)
	}
	if drift1.AuthType != "idp" {
		t.Errorf("unexpected auth_type: got %s, want idp", drift1.AuthType)
	}
	if drift1.AuthUser == nil {
		t.Fatal("expected auth_user to be set")
	}
	if drift1.AuthUser.DisplayName != "John Doe" {
		t.Errorf("unexpected display_name: got %s", drift1.AuthUser.DisplayName)
	}
	if drift1.AuthUser.Position != "DevOps Engineer" {
		t.Errorf("unexpected position: got %s", drift1.AuthUser.Position)
	}
	if drift1.AuthUser.UserPictureURL != "https://example.com/avatar.jpg" {
		t.Errorf("unexpected user_picture_url: got %s", drift1.AuthUser.UserPictureURL)
	}
	if len(drift1.Cmd) != 3 {
		t.Errorf("expected 3 cmd elements, got %d", len(drift1.Cmd))
	}

	// Test second drift with auth_trust
	drift2 := result.Drifts[1]
	if drift2.ID != 201 {
		t.Errorf("unexpected id: got %d, want 201", drift2.ID)
	}
	if drift2.Status != "failed" {
		t.Errorf("unexpected status: got %s, want failed", drift2.Status)
	}
	if drift2.AuthType != "gitlabcicd" {
		t.Errorf("unexpected auth_type: got %s, want gitlabcicd", drift2.AuthType)
	}
	if drift2.AuthTrust == nil {
		t.Fatal("expected auth_trust to be set")
	}
	if drift2.AuthTrust.AuthID != "project/repo/.gitlab-ci.yml@refs/heads/main" {
		t.Errorf("unexpected auth_id: got %s", drift2.AuthTrust.AuthID)
	}
	if drift2.GroupingKey != "repo+id+3" {
		t.Errorf("unexpected grouping_key: got %s", drift2.GroupingKey)
	}

	// Test pagination
	if result.PaginatedResult.Total != 2 {
		t.Errorf("unexpected total: got %d, want 2", result.PaginatedResult.Total)
	}
}

func TestDriftsListForStack_EmptyResponse(t *testing.T) {
	payload := `{
		"paginated_result": {
			"page": 1,
			"per_page": 10,
			"total": 0
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

	result, resp, err := client.Drifts.ListForStack(context.Background(), "org-uuid", 123, nil)
	if err != nil {
		t.Fatalf("ListForStack error: %v", err)
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
	if len(result.Drifts) != 0 {
		t.Errorf("expected 0 drifts, got %d", len(result.Drifts))
	}
	if result.PaginatedResult.Total != 0 {
		t.Errorf("expected total 0, got %d", result.PaginatedResult.Total)
	}
}

func TestDriftsListForStack_HandlesAPIError(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		if _, werr := w.Write([]byte(`{"error":"stack not found"}`)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.Drifts.ListForStack(context.Background(), "org-uuid", 999, nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	if apiErr, ok := err.(*APIError); ok {
		if apiErr.StatusCode != 404 {
			t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
		}
		if apiErr.Message != "stack not found" {
			t.Errorf("unexpected error message: %s", apiErr.Message)
		}
	} else {
		t.Errorf("expected APIError type, got %T", err)
	}
}

func TestDriftsListForStack_SendsAuthHeader(t *testing.T) {
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

		payload := `{"drifts":[],"paginated_result":{"page":1,"per_page":10,"total":0}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.Drifts.ListForStack(context.Background(), "org-uuid", 123, nil)
	if err != nil {
		t.Fatalf("ListForStack error: %v", err)
	}
}

func TestDriftsListForStack_RespectsContextCancellation(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Wait for context cancellation
		<-r.Context().Done()
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err := client.Drifts.ListForStack(ctx, "org-uuid", 123, nil)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestDriftsListForStack_RespectsContextTimeout(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err := client.Drifts.ListForStack(ctx, "org-uuid", 123, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// Get tests

func TestDriftsGet_ParsesResponse(t *testing.T) {
	payload := `{
		"id": 100,
		"org_uuid": "org-uuid-123",
		"stack_id": 456,
		"status": "drifted",
		"metadata": {"key": "value"},
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
			"repository": "github.com/acme/infrastructure",
			"path": "/stacks/vpc",
			"default_branch": "main",
			"meta_id": "vpc-prod-01",
			"status": "ok",
			"deployment_status": "ok",
			"drift_status": "drifted",
			"draft": false,
			"is_archived": false,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-15T12:00:00Z",
			"seen_at": "2024-01-15T12:00:00Z"
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/drifts/org-uuid-123/456/100"
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

	drift, resp, err := client.Drifts.Get(context.Background(), "org-uuid-123", 456, 100)
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

	if drift == nil {
		t.Fatal("expected non-nil drift")
		return
	}
	if drift.ID != 100 {
		t.Errorf("unexpected id: got %d, want 100", drift.ID)
	}
	if drift.OrgUUID != "org-uuid-123" {
		t.Errorf("unexpected org_uuid: got %s", drift.OrgUUID)
	}
	if drift.StackID != 456 {
		t.Errorf("unexpected stack_id: got %d, want 456", drift.StackID)
	}
	if drift.Status != "drifted" {
		t.Errorf("unexpected status: got %s", drift.Status)
	}

	// Verify drift details are populated
	if drift.DriftDetails == nil {
		t.Fatal("expected drift_details to be set")
	}
	if drift.DriftDetails.Provisioner != "terraform" {
		t.Errorf("unexpected provisioner: got %s", drift.DriftDetails.Provisioner)
	}
	if drift.DriftDetails.Serial != 42 {
		t.Errorf("unexpected serial: got %d, want 42", drift.DriftDetails.Serial)
	}
	if drift.DriftDetails.ChangesetAscii != "Terraform will perform the following actions:\n\n  + resource.new\n" {
		t.Errorf("unexpected changeset_ascii: got %s", drift.DriftDetails.ChangesetAscii)
	}
	if drift.DriftDetails.ChangesetJSON != "{\"resource_changes\":[]}" {
		t.Errorf("unexpected changeset_json: got %s", drift.DriftDetails.ChangesetJSON)
	}

	// Verify stack is populated
	if drift.Stack == nil {
		t.Fatal("expected stack to be set")
	}
	if drift.Stack.StackID != 456 {
		t.Errorf("unexpected stack.stack_id: got %d, want 456", drift.Stack.StackID)
	}
	if drift.Stack.MetaID != "vpc-prod-01" {
		t.Errorf("unexpected stack.meta_id: got %s", drift.Stack.MetaID)
	}
}

func TestDriftsGet_Validation(t *testing.T) {
	c, err := NewClient("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tests := []struct {
		name      string
		orgUUID   string
		stackID   int
		driftID   int
		wantError string
	}{
		{"empty org UUID", "", 123, 100, "organization UUID is required"},
		{"zero stack ID", "org-uuid", 0, 100, "stack ID must be positive"},
		{"negative stack ID", "org-uuid", -1, 100, "stack ID must be positive"},
		{"zero drift ID", "org-uuid", 123, 0, "drift ID must be positive"},
		{"negative drift ID", "org-uuid", 123, -1, "drift ID must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.Drifts.Get(context.Background(), tt.orgUUID, tt.stackID, tt.driftID)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantError {
				t.Errorf("got error %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestDriftsGet_HandlesAPIError(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		if _, werr := w.Write([]byte(`{"error":"drift not found"}`)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.Drifts.Get(context.Background(), "org-uuid", 123, 999)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	if apiErr, ok := err.(*APIError); ok {
		if apiErr.StatusCode != 404 {
			t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
		}
		if apiErr.Message != "drift not found" {
			t.Errorf("unexpected error message: %s", apiErr.Message)
		}
	} else {
		t.Errorf("expected APIError type, got %T", err)
	}
}

func TestDriftsGet_SendsAuthHeader(t *testing.T) {
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
			"id": 100,
			"org_uuid": "org-uuid",
			"stack_id": 123,
			"status": "ok",
			"metadata": {}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.Drifts.Get(context.Background(), "org-uuid", 123, 100)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
}

func TestDriftsGet_RespectsContextCancellation(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Wait for context cancellation
		<-r.Context().Done()
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err := client.Drifts.Get(ctx, "org-uuid", 123, 100)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestDriftsGet_RespectsContextTimeout(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err := client.Drifts.Get(ctx, "org-uuid", 123, 100)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
