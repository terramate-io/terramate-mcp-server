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

func TestListDeployments_Success(t *testing.T) {
	payload := `{
		"deployments": [
			{
				"id": 100,
				"status": "ok",
				"commit_title": "feat: Add VPC",
				"repository": "github.com/acme/infra",
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
			"total": 1,
			"page": 1,
			"per_page": 10
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/organizations/org-uuid/deployments" {
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

	tool := ListDeployments(c)
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
	var response terramate.DeploymentsListResponse
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(response.Deployments) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(response.Deployments))
	}
	if response.Deployments[0].ID != 100 {
		t.Fatalf("expected id=100, got %d", response.Deployments[0].ID)
	}
}

func TestListDeployments_MissingOrgUUID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListDeployments(c)
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
}

func TestGetStackDeployment_Success(t *testing.T) {
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

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/stack_deployments/org-uuid/200" {
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

	tool := GetStackDeployment(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid":   "org-uuid",
				"stack_deployment_id": float64(200),
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
	var deployment terramate.StackDeployment
	if err := json.Unmarshal([]byte(textContent.Text), &deployment); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if deployment.ID != 200 {
		t.Fatalf("expected id=200, got %d", deployment.ID)
	}
	if deployment.ChangesetDetails == nil {
		t.Fatal("expected changeset_details to be set")
	}
}

func TestGetStackDeployment_MissingOrgUUID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStackDeployment(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"stack_deployment_id": float64(200),
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

func TestGetStackDeployment_InvalidID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStackDeployment(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid":   "org-uuid",
				"stack_deployment_id": float64(0),
			},
		},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid id")
	}
}
