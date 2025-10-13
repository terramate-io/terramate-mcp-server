package terramate

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient_SetsUserAgentAndAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if !strings.HasPrefix(ua, "terramate-mcp-server/") {
			t.Fatalf("unexpected user agent: %q", ua)
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Fatalf("expected basic auth, got %q", auth)
		}
		decoded, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if !strings.HasSuffix(string(decoded), ":") {
			t.Fatalf("expected empty password in basic auth, got %q", string(decoded))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(`[]`)); werr != nil {
			panic(werr)
		}
	}))
	defer ts.Close()

	c, err := NewClient("test-key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	_, _, err = c.Memberships.List(context.Background())
	if err != nil {
		t.Fatalf("List memberships error: %v", err)
	}
}

func TestWithRegion_SetsExpectedBaseURL(t *testing.T) {
	cEU, err := NewClient("k", WithRegion("eu"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if got := cEU.baseURL.String(); got != "https://api.terramate.io" {
		t.Fatalf("eu baseURL: %s", got)
	}

	cUS, err := NewClient("k", WithRegion("us"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if got := cUS.baseURL.String(); got != "https://api.us.terramate.io" {
		t.Fatalf("us baseURL: %s", got)
	}
}

func TestDo_ParsesAPIErrorJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(400)
		if _, err := w.Write([]byte(`{"error":"bad","details":{"x":1}}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()
	c, err := NewClient("key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	req, err := c.newRequest(context.Background(), http.MethodGet, "/x", nil)
	if err != nil {
		t.Fatalf("newRequest: %v", err)
	}
	_, err = c.do(req, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Message != "bad" || apiErr.StatusCode != 400 {
		t.Fatalf("unexpected apiErr: %#v", err)
	}
}

func TestDo_Handles204NoContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	defer ts.Close()
	c, err := NewClient("key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	req, err := c.newRequest(context.Background(), http.MethodGet, "/x", nil)
	if err != nil {
		t.Fatalf("newRequest: %v", err)
	}
	_, err = c.do(req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
