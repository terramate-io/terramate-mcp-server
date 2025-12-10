package terramate

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// testJWTCredential is a test helper that allows mocking the Refresh method
type testJWTCredential struct {
	JWTCredential
	onRefresh func()
	newToken  string
}

func (t *testJWTCredential) Refresh(ctx context.Context) error {
	if t.onRefresh != nil {
		t.onRefresh()
	}
	t.mu.Lock()
	t.idToken = t.newToken
	t.mu.Unlock()
	return nil
}

func TestClient_401RetryWithRefresh(t *testing.T) {
	t.Run("successfully refreshes and retries on 401", func(t *testing.T) {
		testSuccessfulRefreshAndRetry(t)
	})

	t.Run("does not retry with API key", func(t *testing.T) {
		requestCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		}))
		defer server.Close()

		// Create client with API key (not JWT)
		client, err := NewClientWithAPIKey("test-api-key", WithBaseURL(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// Make request
		req, _ := client.newRequest(context.Background(), "GET", "/test", nil)
		_, err = client.do(req, nil)

		if err == nil {
			t.Fatal("expected error for 401 with API key")
		}

		if requestCount != 1 {
			t.Errorf("expected only 1 request (no retry for API key), got %d", requestCount)
		}

		t.Log("✓ API key auth does not attempt refresh on 401")
	})

	t.Run("handles refresh failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		}))
		defer server.Close()

		// Create credential without refresh token
		cred := &JWTCredential{
			idToken:  "old-expired-token",
			provider: "Google",
			// No refresh token
		}

		client, err := NewClient(cred, WithBaseURL(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// Make request
		req, _ := client.newRequest(context.Background(), "GET", "/test", nil)
		_, err = client.do(req, nil)

		if err == nil {
			t.Fatal("expected error when refresh fails")
		}

		// Should get the original 401 error since refresh failed
		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected APIError, got %T", err)
		}

		if apiErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", apiErr.StatusCode)
		}

		t.Log("✓ Returns 401 error when refresh fails")
	})

	t.Run("prevents unbounded recursion on repeated 401", func(t *testing.T) {
		requestCount := 0
		refreshCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			// Always return 401, even after refresh
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		}))
		defer server.Close()

		cred := &testJWTCredential{
			JWTCredential: JWTCredential{
				idToken:      "old-expired-token",
				refreshToken: "refresh-token",
				provider:     "Google",
			},
			onRefresh: func() {
				refreshCount++
			},
			newToken: "new-token",
		}

		client, err := NewClient(cred, WithBaseURL(server.URL))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// Make request
		req, _ := client.newRequest(context.Background(), "GET", "/test", nil)
		_, err = client.do(req, nil)

		if err == nil {
			t.Fatal("expected error for repeated 401")
		}

		// Should get 401 error
		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected APIError, got %T", err)
		}

		if apiErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", apiErr.StatusCode)
		}

		// Should have attempted refresh once, then stopped
		if refreshCount != 1 {
			t.Errorf("expected 1 refresh attempt, got %d", refreshCount)
		}

		// Should have made initial request + 1 retry = 2 requests max
		if requestCount > 2 {
			t.Errorf("expected at most 2 requests (initial + 1 retry), got %d", requestCount)
		}

		t.Logf("✓ Prevented unbounded recursion: %d refresh attempts, %d total requests", refreshCount, requestCount)
	})
}

func TestClient_ConcurrentRequestsDuringRefresh(t *testing.T) {
	var mu sync.Mutex
	refreshCount := 0
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		// Always return success
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cred := &testJWTCredential{
		JWTCredential: JWTCredential{
			idToken:      "test-token",
			refreshToken: "refresh-token",
			provider:     "Google",
		},
		onRefresh: func() {
			mu.Lock()
			refreshCount++
			mu.Unlock()
		},
		newToken: "new-token",
	}

	client, err := NewClient(cred, WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Make multiple concurrent requests
	errors := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req, _ := client.newRequest(context.Background(), "GET", "/test", nil)
			_, err := client.do(req, nil)
			errors <- err
		}()
	}

	// Collect errors
	for i := 0; i < 10; i++ {
		if err := <-errors; err != nil {
			t.Errorf("request %d failed: %v", i, err)
		}
	}

	mu.Lock()
	finalRefreshCount := refreshCount
	finalRequestCount := requestCount
	mu.Unlock()

	t.Logf("Refresh called %d times for 10 concurrent requests", finalRefreshCount)
	t.Logf("Server received %d total requests", finalRequestCount)

	t.Log("✓ Concurrent requests handled successfully")
}

// testSuccessfulRefreshAndRetry tests the happy path: 401 -> refresh -> retry -> success
func testSuccessfulRefreshAndRetry(t *testing.T) {
	requestCount := 0
	refreshCount := 0
	oldToken := "old-expired-token"
	newToken := "new-refreshed-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		authHeader := r.Header.Get("Authorization")

		// First request: return 401 with old token
		if requestCount == 1 {
			if authHeader != "Bearer "+oldToken {
				t.Errorf("first request: expected Authorization header with old token, got %q", authHeader)
			}
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		// Second request (after refresh): return 200 with new token
		if requestCount == 2 {
			if authHeader != "Bearer "+newToken {
				t.Errorf("retry request: expected Authorization header with new token, got %q", authHeader)
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		// Should not reach here
		t.Errorf("unexpected request count: %d", requestCount)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cred := &testJWTCredential{
		JWTCredential: JWTCredential{
			idToken:      oldToken,
			refreshToken: "refresh-token",
			provider:     "Google",
		},
		onRefresh: func() {
			refreshCount++
		},
		newToken: newToken,
	}

	client, err := NewClient(cred, WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Make request - should trigger refresh and retry
	req, _ := client.newRequest(context.Background(), "GET", "/test", nil)
	resp, err := client.do(req, nil)
	if err != nil {
		t.Fatalf("expected success after refresh, got error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.HTTPResponse.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.HTTPResponse.StatusCode)
	}

	// Verify refresh was called exactly once
	if refreshCount != 1 {
		t.Errorf("expected 1 refresh call, got %d", refreshCount)
	}

	// Verify exactly 2 requests were made (initial + retry)
	if requestCount != 2 {
		t.Errorf("expected 2 requests (initial + retry), got %d", requestCount)
	}

	t.Log("✓ Successfully refreshed token and retried request on 401")
}

// customReader is a custom io.Reader type that doesn't have GetBody set automatically
type customReader struct {
	data   []byte
	offset int
}

func (r *customReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}

// testBodyReader401Retry is a helper function to test 401 retry with different body reader types
func testBodyReader401Retry(t *testing.T, name string, bodyReader io.Reader, requestBody string, tokenSuffix string) {
	t.Helper()
	requestCount := 0
	refreshCount := 0
	oldToken := "old-expired-token-" + tokenSuffix
	newToken := "new-refreshed-token-" + tokenSuffix

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		authHeader := r.Header.Get("Authorization")

		// Read and verify request body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if string(bodyBytes) != requestBody {
			t.Errorf("request %d: expected body %q, got %q", requestCount, requestBody, string(bodyBytes))
		}

		// First request: return 401 with old token
		if requestCount == 1 {
			if authHeader != "Bearer "+oldToken {
				t.Errorf("first request: expected Authorization header with old token, got %q", authHeader)
			}
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		// Second request (after refresh): return 200 with new token
		if requestCount == 2 {
			if authHeader != "Bearer "+newToken {
				t.Errorf("retry request: expected Authorization header with new token, got %q", authHeader)
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		// Should not reach here
		t.Errorf("unexpected request count: %d", requestCount)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cred := &testJWTCredential{
		JWTCredential: JWTCredential{
			idToken:      oldToken,
			refreshToken: "refresh-token",
			provider:     "Google",
		},
		onRefresh: func() {
			refreshCount++
		},
		newToken: newToken,
	}

	client, err := NewClient(cred, WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req, err := client.newRequest(context.Background(), "POST", "/test", bodyReader)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Verify GetBody is set
	if req.GetBody == nil {
		t.Fatalf("expected GetBody to be set for %s", name)
	}

	resp, err := client.do(req, nil)
	if err != nil {
		t.Fatalf("expected success after refresh, got error: %v", err)
	}

	if resp.HTTPResponse.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.HTTPResponse.StatusCode)
	}

	if refreshCount != 1 {
		t.Errorf("expected 1 refresh call, got %d", refreshCount)
	}

	if requestCount != 2 {
		t.Errorf("expected 2 requests (initial + retry), got %d", requestCount)
	}
}

func TestClient_401RetryWithCustomBodyReader(t *testing.T) {
	requestBody := `{"test": "data"}`

	t.Run("with strings.Reader", func(t *testing.T) {
		testBodyReader401Retry(t, "strings.Reader", strings.NewReader(requestBody), requestBody, "1")
	})

	t.Run("with custom io.Reader", func(t *testing.T) {
		testBodyReader401Retry(t, "custom io.Reader", &customReader{data: []byte(requestBody)}, requestBody, "2")
	})

	t.Run("with bytes.Buffer", func(t *testing.T) {
		testBodyReader401Retry(t, "bytes.Buffer", bytes.NewBufferString(requestBody), requestBody, "3")
	})
}
