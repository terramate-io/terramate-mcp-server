package tmc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

func TestListStacks_Success(t *testing.T) {
	payload := `{
		"stacks": [
			{
				"stack_id": 1,
				"repository": "github.com/acme/infra",
				"path": "/stacks/vpc",
				"default_branch": "main",
				"meta_id": "vpc",
				"meta_name": "VPC Stack",
				"meta_description": "Main VPC infrastructure",
				"meta_tags": ["network", "production"],
				"status": "ok",
				"deployment_status": "deployed",
				"drift_status": "ok",
				"draft": false,
				"is_archived": false,
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-02T00:00:00Z"
			}
		],
		"paginated_result": {
			"total": 1,
			"page": 1,
			"per_page": 20
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/stacks/org-uuid" {
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

	tool := ListStacks(c)
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
	var response terramate.StacksListResponse
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(response.Stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(response.Stacks))
	}
	if response.Stacks[0].StackID != 1 {
		t.Fatalf("expected stack_id=1, got %d", response.Stacks[0].StackID)
	}
	if response.PaginatedResult.Total != 1 {
		t.Fatalf("expected total=1, got %d", response.PaginatedResult.Total)
	}
}

func TestListStacks_WithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("search") != "vpc" {
			t.Errorf("expected search=vpc, got %s", query.Get("search"))
		}
		if query.Get("status") != "ok,failed" {
			t.Errorf("expected status=ok,failed, got %s", query.Get("status"))
		}
		if query.Get("page") != "2" {
			t.Errorf("expected page=2, got %s", query.Get("page"))
		}
		if query.Get("per_page") != "10" {
			t.Errorf("expected per_page=10, got %s", query.Get("per_page"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(`{"stacks":[],"paginated_result":{"total":0,"page":2,"per_page":10}}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListStacks(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"search":            "vpc",
				"status":            []interface{}{"ok", "failed"},
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

func TestListStacks_MissingOrgUUID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListStacks(c)
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

func TestListStacks_WithDraftFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("draft") != "true" {
			t.Errorf("expected draft=true, got %s", query.Get("draft"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(`{"stacks":[],"paginated_result":{"total":0,"page":1,"per_page":20}}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListStacks(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"draft":             true,
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

func TestListStacks_InvalidPerPage(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := ListStacks(c)
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
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "Per page value must not exceed 100." {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestListStacks_Unauthorized(t *testing.T) {
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

	tool := ListStacks(c)
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

func TestGetStack_Success(t *testing.T) {
	payload := `{
		"stack_id": 42,
		"repository": "github.com/acme/infra",
		"path": "/stacks/vpc",
		"default_branch": "main",
		"meta_id": "vpc",
		"meta_name": "VPC Stack",
		"meta_description": "Main VPC infrastructure",
		"meta_tags": ["network", "production"],
		"status": "ok",
		"deployment_status": "deployed",
		"drift_status": "ok",
		"draft": false,
		"is_archived": false,
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-02T00:00:00Z",
		"related_stacks": [
			{
				"stack_id": 43,
				"target": "staging"
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/stacks/org-uuid/42" {
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

	tool := GetStack(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(42),
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
	var stack terramate.Stack
	if err := json.Unmarshal([]byte(textContent.Text), &stack); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if stack.StackID != 42 {
		t.Fatalf("expected stack_id=42, got %d", stack.StackID)
	}
	if stack.MetaName != "VPC Stack" {
		t.Fatalf("expected meta_name='VPC Stack', got %s", stack.MetaName)
	}
	if len(stack.RelatedStacks) != 1 {
		t.Fatalf("expected 1 related stack, got %d", len(stack.RelatedStacks))
	}
}

func TestGetStack_MissingOrgUUID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStack(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"stack_id": float64(42),
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

func TestGetStack_MissingStackID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStack(c)
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

func TestGetStack_InvalidStackID(t *testing.T) {
	c, err := terramate.NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := GetStack(c)
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

func TestGetStack_NotFound(t *testing.T) {
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

	tool := GetStack(c)
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

func TestGetStack_Unauthorized(t *testing.T) {
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

	tool := GetStack(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(42),
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

// archiveToolCase describes one of the two archive-state tools under test.
type archiveToolCase struct {
	name     string
	newTool  func(*terramate.Client) server.ServerTool
	archived bool
	verb     string
}

var archiveToolCases = []archiveToolCase{
	{name: "tmc_archive_stack", newTool: ArchiveStack, archived: true, verb: "archive"},
	{name: "tmc_unarchive_stack", newTool: UnarchiveStack, archived: false, verb: "unarchive"},
}

func callArchiveTool(t *testing.T, tc archiveToolCase, handler http.HandlerFunc, args map[string]interface{}) (*mcp.CallToolResult, string) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	result, err := tc.newTool(c).Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: args},
	})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	return result, textContent.Text
}

func TestArchiveTools_Success(t *testing.T) {
	for _, tc := range archiveToolCases {
		t.Run(tc.name, func(t *testing.T) {
			var patched bool
			handler := func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/stacks/org-uuid/42" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				switch r.Method {
				case http.MethodPatch:
					var body map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("failed to decode body: %v", err)
					}
					if body["is_archived"] != tc.archived {
						t.Errorf("unexpected is_archived in body: %v", body["is_archived"])
					}
					patched = true
					w.WriteHeader(http.StatusNoContent)
				case http.MethodGet:
					if !patched {
						t.Error("stack fetched before it was updated")
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprintf(w, `{"stack_id": 42, "path": "/stacks/vpc", "is_archived": %t}`, tc.archived)
				default:
					t.Errorf("unexpected method: %s", r.Method)
				}
			}

			result, text := callArchiveTool(t, tc, handler, map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(42),
			})
			if result.IsError {
				t.Fatalf("unexpected error result: %s", text)
			}
			wantPrefix := fmt.Sprintf("Stack 42 %sd.\n\n", tc.verb)
			if !strings.HasPrefix(text, wantPrefix) {
				t.Fatalf("expected prefix %q, got %q", wantPrefix, text)
			}
			var stack terramate.Stack
			if err := json.Unmarshal([]byte(strings.TrimPrefix(text, wantPrefix)), &stack); err != nil {
				t.Fatalf("failed to parse stack JSON: %v", err)
			}
			if stack.StackID != 42 || stack.IsArchived != tc.archived {
				t.Errorf("unexpected stack in result: %+v", stack)
			}
		})
	}
}

func TestArchiveTools_InvalidInput(t *testing.T) {
	inputs := []struct {
		name    string
		args    map[string]interface{}
		wantMsg string
	}{
		{"missing org UUID", map[string]interface{}{"stack_id": float64(42)}, "Organization UUID is required and must be a string."},
		{"missing stack ID", map[string]interface{}{"organization_uuid": "org-uuid"}, "Stack ID is required and must be a number."},
		{"non-positive stack ID", map[string]interface{}{"organization_uuid": "org-uuid", "stack_id": float64(0)}, "Stack ID must be positive."},
	}

	for _, tc := range archiveToolCases {
		for _, in := range inputs {
			t.Run(tc.name+"/"+in.name, func(t *testing.T) {
				handler := func(w http.ResponseWriter, r *http.Request) {
					t.Error("no request should be sent for invalid input")
				}
				result, text := callArchiveTool(t, tc, handler, in.args)
				if !result.IsError || text != in.wantMsg {
					t.Fatalf("expected error %q, got isError=%v text=%q", in.wantMsg, result.IsError, text)
				}
			})
		}
	}
}

func TestArchiveTools_APIErrors(t *testing.T) {
	for _, tc := range archiveToolCases {
		errs := []struct {
			status  int
			wantMsg string
		}{
			{http.StatusUnauthorized, terramate.ErrAuthenticationFailed},
			{http.StatusForbidden, fmt.Sprintf("Permission denied: you are not allowed to %s stack 42.", tc.verb)},
			{http.StatusNotFound, "Stack with ID 42 not found."},
		}
		for _, e := range errs {
			t.Run(fmt.Sprintf("%s/%d", tc.name, e.status), func(t *testing.T) {
				handler := func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPatch {
						t.Errorf("expected only the PATCH request, got %s", r.Method)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(e.status)
					_, _ = w.Write([]byte(`{"error":"nope"}`))
				}
				result, text := callArchiveTool(t, tc, handler, map[string]interface{}{
					"organization_uuid": "org-uuid",
					"stack_id":          float64(42),
				})
				if !result.IsError || text != e.wantMsg {
					t.Fatalf("expected error %q, got isError=%v text=%q", e.wantMsg, result.IsError, text)
				}
			})
		}
	}
}

// If the update succeeds but re-reading the stack fails, the tool must still
// report success: the change was applied and must not be presented as failed.
func TestArchiveTools_FollowUpGetFails(t *testing.T) {
	for _, tc := range archiveToolCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPatch {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}
			result, text := callArchiveTool(t, tc, handler, map[string]interface{}{
				"organization_uuid": "org-uuid",
				"stack_id":          float64(42),
			})
			if result.IsError {
				t.Fatalf("expected success result, got error: %s", text)
			}
			wantPrefix := fmt.Sprintf("Stack 42 %sd, but fetching its updated details failed:", tc.verb)
			if !strings.HasPrefix(text, wantPrefix) {
				t.Fatalf("expected prefix %q, got %q", wantPrefix, text)
			}
		})
	}
}

func TestArchiveTools_Annotations(t *testing.T) {
	for _, tc := range archiveToolCases {
		t.Run(tc.name, func(t *testing.T) {
			tool := tc.newTool(nil).Tool
			if tool.Name != tc.name {
				t.Errorf("unexpected tool name: %s", tool.Name)
			}
			a := tool.Annotations
			if a.ReadOnlyHint == nil || *a.ReadOnlyHint {
				t.Error("expected readOnlyHint=false")
			}
			if a.DestructiveHint == nil || *a.DestructiveHint {
				t.Error("expected destructiveHint=false (archiving is reversible)")
			}
			if a.IdempotentHint == nil || !*a.IdempotentHint {
				t.Error("expected idempotentHint=true")
			}
			if a.OpenWorldHint == nil || !*a.OpenWorldHint {
				t.Error("expected openWorldHint=true")
			}
		})
	}
}
