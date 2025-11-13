package terramate

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestDeploymentsList_ParsesResponse(t *testing.T) {
	payload := `{
		"deployments": [
			{
				"id": 100,
				"status": "ok",
				"commit_title": "feat: Add VPC",
				"repository": "github.com/acme/infrastructure",
				"canceled_count": 0,
				"failed_count": 0,
				"ok_count": 5,
				"pending_count": 0,
				"running_count": 0,
				"stack_deployment_total_count": 5,
				"created_at": "2024-01-15T10:00:00Z"
			}
		],
		"paginated_result": {
			"page": 1,
			"per_page": 10,
			"total": 1
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/organizations/org-uuid-123/deployments"
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

	result, resp, err := client.Deployments.List(context.Background(), "org-uuid-123", nil)
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
	if len(result.Deployments) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(result.Deployments))
	}

	deployment := result.Deployments[0]
	if deployment.ID != 100 {
		t.Errorf("unexpected id: got %d, want 100", deployment.ID)
	}
	if deployment.Status != "ok" {
		t.Errorf("unexpected status: got %s", deployment.Status)
	}
	if deployment.OkCount != 5 {
		t.Errorf("unexpected ok_count: got %d, want 5", deployment.OkCount)
	}
}

func TestDeploymentsList_WithOptions(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("repository") != "github.com/acme/repo" {
			t.Errorf("expected repository=github.com/acme/repo, got %s", query.Get("repository"))
		}
		if query.Get("status") != "ok,failed" {
			t.Errorf("expected status=ok,failed, got %s", query.Get("status"))
		}
		if query.Get("auth_type") != "github,gitlab" {
			t.Errorf("expected auth_type=github,gitlab, got %s", query.Get("auth_type"))
		}

		payload := `{"deployments":[],"paginated_result":{"page":1,"per_page":10,"total":0}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	opts := &DeploymentsListOptions{
		Repository: []string{"github.com/acme/repo"},
		Status:     []string{"ok", "failed"},
		AuthType:   []string{"github", "gitlab"},
	}

	_, _, err := client.Deployments.List(context.Background(), "org-uuid", opts)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
}

func TestDeploymentsList_Validation(t *testing.T) {
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
			_, _, err := c.Deployments.List(context.Background(), tt.orgUUID, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantError {
				t.Errorf("got error %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestDeploymentsGetWorkflow_ParsesResponse(t *testing.T) {
	payload := `{
		"id": 100,
		"status": "ok",
		"commit_title": "feat: Add VPC",
		"repository": "github.com/acme/infrastructure",
		"canceled_count": 0,
		"failed_count": 0,
		"ok_count": 5,
		"pending_count": 0,
		"running_count": 0,
		"stack_deployment_total_count": 5,
		"created_at": "2024-01-15T10:00:00Z"
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/workflow_deployment_groups/org-uuid/100"
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

	workflow, resp, err := client.Deployments.GetWorkflow(context.Background(), "org-uuid", 100)
	if err != nil {
		t.Fatalf("GetWorkflow error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if workflow == nil {
		t.Fatal("expected non-nil workflow")
		return
	}
	if workflow.ID != 100 {
		t.Errorf("unexpected id: got %d, want 100", workflow.ID)
	}
}

func TestDeploymentsGetWorkflow_Validation(t *testing.T) {
	c, err := NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tests := []struct {
		name                      string
		orgUUID                   string
		workflowDeploymentGroupID int
		wantError                 string
	}{
		{"empty org UUID", "", 100, "organization UUID is required"},
		{"zero workflow ID", "org-uuid", 0, "workflow deployment group ID must be positive"},
		{"negative workflow ID", "org-uuid", -1, "workflow deployment group ID must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.Deployments.GetWorkflow(context.Background(), tt.orgUUID, tt.workflowDeploymentGroupID)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantError {
				t.Errorf("got error %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestDeploymentsListStackDeployments_ParsesResponse(t *testing.T) {
	payload := `{
		"stack_deployments": [
			{
				"id": 200,
				"deployment_uuid": "deploy-uuid-123",
				"path": "/stacks/vpc",
				"cmd": ["terraform", "apply"],
				"status": "ok",
				"created_at": "2024-01-15T10:00:00Z"
			}
		],
		"paginated_result": {
			"page": 1,
			"per_page": 10,
			"total": 1
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/stack_deployments/org-uuid"
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

	result, resp, err := client.Deployments.ListStackDeployments(context.Background(), "org-uuid", nil)
	if err != nil {
		t.Fatalf("ListStackDeployments error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if len(result.StackDeployments) != 1 {
		t.Fatalf("expected 1 stack deployment, got %d", len(result.StackDeployments))
	}
	if result.StackDeployments[0].ID != 200 {
		t.Errorf("unexpected id: got %d, want 200", result.StackDeployments[0].ID)
	}
}

func TestDeploymentsGetStackDeployment_ParsesResponse(t *testing.T) {
	payload := `{
		"id": 200,
		"deployment_uuid": "deploy-uuid-123",
		"path": "/stacks/vpc",
		"cmd": ["terraform", "apply"],
		"status": "ok",
		"created_at": "2024-01-15T10:00:00Z",
		"changeset_details": {
			"provisioner": "terraform",
			"changeset_ascii": "Terraform plan output..."
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/stack_deployments/org-uuid/200"
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

	deployment, resp, err := client.Deployments.GetStackDeployment(context.Background(), "org-uuid", 200)
	if err != nil {
		t.Fatalf("GetStackDeployment error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if deployment == nil {
		t.Fatal("expected non-nil deployment")
		return
	}
	if deployment.ID != 200 {
		t.Errorf("unexpected id: got %d, want 200", deployment.ID)
	}
	if deployment.ChangesetDetails == nil {
		t.Fatal("expected changeset_details to be set")
	}
}

func TestDeploymentsGetStackDeployment_Validation(t *testing.T) {
	c, err := NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tests := []struct {
		name              string
		orgUUID           string
		stackDeploymentID int
		wantError         string
	}{
		{"empty org UUID", "", 200, "organization UUID is required"},
		{"zero deployment ID", "org-uuid", 0, "stack deployment ID must be positive"},
		{"negative deployment ID", "org-uuid", -1, "stack deployment ID must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.Deployments.GetStackDeployment(context.Background(), tt.orgUUID, tt.stackDeploymentID)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantError {
				t.Errorf("got error %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestDeploymentsList_RespectsContextCancellation(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := client.Deployments.List(ctx, "org-uuid", nil)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestDeploymentsGetWorkflow_RespectsContextTimeout(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err := client.Deployments.GetWorkflow(ctx, "org-uuid", 100)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
