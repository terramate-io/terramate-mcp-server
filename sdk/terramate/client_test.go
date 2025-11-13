package terramate

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

	c, err := NewClientWithAPIKey("test-key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	_, _, err = c.Memberships.List(context.Background())
	if err != nil {
		t.Fatalf("List memberships error: %v", err)
	}
}

func TestWithRegion_SetsExpectedBaseURL(t *testing.T) {
	cEU, err := NewClientWithAPIKey("k", WithRegion("eu"))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if got := cEU.baseURL.String(); got != "https://api.terramate.io" {
		t.Fatalf("eu baseURL: %s", got)
	}

	cUS, err := NewClientWithAPIKey("k", WithRegion("us"))
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
	c, err := NewClientWithAPIKey("key", WithBaseURL(ts.URL))
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
	c, err := NewClientWithAPIKey("key", WithBaseURL(ts.URL))
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

func TestNewClientWithJWT_SetsBearerAuth(t *testing.T) {
	jwtToken := generateTestJWT(time.Now().Add(1 * time.Hour))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + jwtToken
		if auth != expected {
			t.Fatalf("expected Bearer auth with JWT, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(`[]`)); werr != nil {
			panic(werr)
		}
	}))
	defer ts.Close()

	c, err := NewClientWithJWT(jwtToken, WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClientWithJWT error: %v", err)
	}

	_, _, err = c.Memberships.List(context.Background())
	if err != nil {
		t.Fatalf("List memberships error: %v", err)
	}
}

func TestNewClientWithJWT_ExpiredTokenSentToAPI(t *testing.T) {
	// Create an expired JWT token - client should still send it to the API
	expiredToken := generateTestJWT(time.Now().Add(-1 * time.Hour))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the Authorization header is set even for "expired" token
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+expiredToken {
			t.Errorf("Expected Bearer token in header, got: %v", authHeader)
		}

		// Simulate API rejecting expired token with 401
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"Invalid or expired token"}`))
	}))
	defer ts.Close()

	c, err := NewClientWithJWT(expiredToken, WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClientWithJWT error: %v", err)
	}

	// Make a request - client sends token, API rejects it with 401
	_, _, err = c.Memberships.List(context.Background())
	if err == nil {
		t.Fatal("expected error from API for expired token")
	}

	// Check if the error message provides helpful guidance
	errMsg := err.Error()
	if !strings.Contains(errMsg, "Authentication failed") {
		t.Errorf("expected 'Authentication failed' in error message, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "terramate cloud login") {
		t.Errorf("expected error to mention 'terramate cloud login', got: %v", errMsg)
	}
}
