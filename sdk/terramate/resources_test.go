package terramate

import (
	"context"
	"net/http"
	"testing"
)

func TestResourcesList_ParsesResponse(t *testing.T) {
	payload := `{
		"resources": [
			{
				"resource_uuid": "f1c9ecfe-1a45-499b-ab6d-1aa0a8ea2f95",
				"stack": {
					"stack_id": 1,
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
					"updated_at": "2024-01-02T00:00:00Z"
				},
				"provisioner": "terraform",
				"descriptor": {
					"address": "aws_vpc.main",
					"type": "aws_vpc",
					"provider_name": "aws",
					"extracted_id": "vpc-123",
					"schema_version": 0
				},
				"status": "ok",
				"drifted": false,
				"pending": false,
				"created_at": "2024-04-12T07:06:00Z",
				"updated_at": "2024-04-15T11:05:00Z"
			}
		],
		"paginated_result": {
			"total": 1,
			"page": 1,
			"per_page": 20
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/resources/org-uuid-123" {
			t.Errorf("unexpected path: got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	result, resp, err := client.Resources.List(context.Background(), "org-uuid-123", nil)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if resp == nil || resp.HTTPResponse.StatusCode != 200 {
		t.Fatalf("expected status 200, got %v", resp)
	}
	if result == nil || len(result.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(result.Resources))
	}

	res := result.Resources[0]
	if res.ResourceUUID != "f1c9ecfe-1a45-499b-ab6d-1aa0a8ea2f95" {
		t.Errorf("unexpected resource_uuid: %s", res.ResourceUUID)
	}
	if res.Provisioner != "terraform" {
		t.Errorf("unexpected provisioner: %s", res.Provisioner)
	}
	if res.Descriptor.Address != "aws_vpc.main" || res.Descriptor.Type != "aws_vpc" {
		t.Errorf("unexpected descriptor: %+v", res.Descriptor)
	}
	if res.Stack.StackID != 1 || res.Stack.MetaID != "vpc" {
		t.Errorf("unexpected stack: %+v", res.Stack)
	}
	if result.PaginatedResult.Total != 1 || result.PaginatedResult.Page != 1 {
		t.Errorf("unexpected pagination: %+v", result.PaginatedResult)
	}
}

func TestResourcesList_QueryParams(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("stack_id") != "42" {
			t.Errorf("expected stack_id=42, got %s", q.Get("stack_id"))
		}
		if q.Get("status") != "ok,drifted" {
			t.Errorf("expected status=ok,drifted, got %s", q.Get("status"))
		}
		if q.Get("technology") != "terraform" {
			t.Errorf("expected technology=terraform, got %s", q.Get("technology"))
		}
		if q.Get("type") != "aws_vpc" {
			t.Errorf("expected type=aws_vpc, got %s", q.Get("type"))
		}
		if q.Get("search") != "vpc" {
			t.Errorf("expected search=vpc, got %s", q.Get("search"))
		}
		if q.Get("page") != "2" || q.Get("per_page") != "50" {
			t.Errorf("expected page=2 per_page=50, got page=%s per_page=%s", q.Get("page"), q.Get("per_page"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"resources":[],"paginated_result":{"total":0,"page":2,"per_page":50}}`))
	})
	defer cleanup()

	opts := &ResourcesListOptions{
		ListOptions:     ListOptions{Page: 2, PerPage: 50},
		StackID:         42,
		Status:          []string{"ok", "drifted"},
		Technology:      []string{"terraform"},
		Type:            []string{"aws_vpc"},
		Search:          "vpc",
	}
	_, _, err := client.Resources.List(context.Background(), "org-uuid", opts)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
}

func TestResourcesList_OrgUUIDRequired(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer cleanup()

	_, _, err := client.Resources.List(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error when org UUID is empty")
	}
}

func TestResourcesGet_ParsesResponse(t *testing.T) {
	payload := `{
		"resource_uuid": "f1c9ecfe-1a45-499b-ab6d-1aa0a8ea2f95",
		"stack": {
			"stack_id": 1,
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
			"updated_at": "2024-01-02T00:00:00Z"
		},
		"descriptor": {
			"address": "aws_vpc.main",
			"type": "aws_vpc",
			"provider_name": "aws",
			"schema_version": 0
		},
		"status": "ok",
		"drifted": false,
		"pending": false,
		"created_at": "2024-04-12T07:06:00Z",
		"updated_at": "2024-04-15T11:05:00Z",
		"details": {
			"values": "{\"id\":\"vpc-123\"}"
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/resources/org-uuid/f1c9ecfe-1a45-499b-ab6d-1aa0a8ea2f95" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(payload))
	})
	defer cleanup()

	resource, resp, err := client.Resources.Get(context.Background(), "org-uuid", "f1c9ecfe-1a45-499b-ab6d-1aa0a8ea2f95")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if resp == nil || resp.HTTPResponse.StatusCode != 200 {
		t.Fatalf("expected status 200")
	}
	if resource.ResourceUUID != "f1c9ecfe-1a45-499b-ab6d-1aa0a8ea2f95" {
		t.Errorf("unexpected resource_uuid: %s", resource.ResourceUUID)
	}
	if resource.Details == nil || resource.Details.Values != "{\"id\":\"vpc-123\"}" {
		t.Errorf("unexpected details: %+v", resource.Details)
	}
}

func TestResourcesGet_ValidatesInput(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {})
	defer cleanup()

	_, _, err := client.Resources.Get(context.Background(), "", "uuid")
	if err == nil {
		t.Error("expected error when org UUID is empty")
	}

	_, _, err = client.Resources.Get(context.Background(), "org-uuid", "")
	if err == nil {
		t.Error("expected error when resource UUID is empty")
	}
}
