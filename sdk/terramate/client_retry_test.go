package terramate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_RetriesOn429(t *testing.T) {
	attempts := atomic.Int32{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(`[]`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := NewClient("key", WithBaseURL(ts.URL), WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, _, err = c.Memberships.List(context.Background())
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts.Load() < 3 {
		t.Fatalf("expected at least 3 attempts, got: %d", attempts.Load())
	}
}

func TestClient_RetriesOn500(t *testing.T) {
	attempts := atomic.Int32{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(`[]`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := NewClient("key", WithBaseURL(ts.URL), WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, _, err = c.Memberships.List(context.Background())
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts.Load() < 2 {
		t.Fatalf("expected at least 2 attempts, got: %d", attempts.Load())
	}
}

func TestClient_NoRetryOn400(t *testing.T) {
	attempts := atomic.Int32{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"error":"bad"}`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	c, err := NewClient("key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, _, err = c.Memberships.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if attempts.Load() != 1 {
		t.Fatalf("expected exactly 1 attempt for 400, got: %d", attempts.Load())
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	c, err := NewClient("key", WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err = c.Memberships.List(ctx)
	if err == nil {
		t.Fatal("expected context timeout error")
	}
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 1 * time.Second}
	c, err := NewClient("key", WithHTTPClient(customClient))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if c.httpClient.Timeout != 1*time.Second {
		t.Fatalf("expected custom timeout, got: %v", c.httpClient.Timeout)
	}
}

func TestWithHTTPClient_NilError(t *testing.T) {
	_, err := NewClient("key", WithHTTPClient(nil))
	if err == nil || err.Error() != "failed to apply client option: HTTP client cannot be nil" {
		t.Fatalf("expected nil HTTP client error, got: %v", err)
	}
}

func TestWithTimeout(t *testing.T) {
	c, err := NewClient("key", WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	if c.httpClient.Timeout != 5*time.Second {
		t.Fatalf("expected 5s timeout, got: %v", c.httpClient.Timeout)
	}
}

func TestNewClient_EmptyAPIKey(t *testing.T) {
	_, err := NewClient("")
	if err == nil || err.Error() != "API key is required" {
		t.Fatalf("expected API key required error, got: %v", err)
	}
}

func TestWithRegion_InvalidRegion(t *testing.T) {
	_, err := NewClient("key", WithRegion("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid region")
	}
}

func TestWithBaseURL_InvalidURL(t *testing.T) {
	_, err := NewClient("key", WithBaseURL("://invalid"))
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}
