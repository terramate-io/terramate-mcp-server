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

func TestAuthenticate_Success(t *testing.T) {
	payload := `[{"member_id":123,"org_uuid":"org-uuid","org_name":"acme","org_display_name":"Acme Inc","org_domain":"acme.example","role":"admin","status":"active"}]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	tool := Authenticate(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{})
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
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["authenticated"] != true {
		t.Fatalf("expected authenticated=true, got: %v", response)
	}
	if response["organization_uuid"] != "org-uuid" {
		t.Fatalf("expected org_uuid, got: %v", response)
	}
}

func TestAuthenticate_Unauthorized(t *testing.T) {
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

	tool := Authenticate(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected error result for 401")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != terramate.ErrAuthenticationFailed {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestAuthenticate_NoMemberships(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(`[]`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := Authenticate(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected error result for empty memberships")
	}
	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatal("expected TextContent")
	}
	if textContent.Text != "No organization memberships found for this API key" {
		t.Fatalf("unexpected error message: %s", textContent.Text)
	}
}

func TestAuthenticate_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		if _, err := w.Write([]byte(`{"error":"internal error"}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := terramate.NewClientWithAPIKey("key", terramate.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tool := Authenticate(c)
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected error result for 500")
	}
}
