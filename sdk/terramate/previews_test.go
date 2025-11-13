package terramate

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestPreviewsGet_ParsesResponse(t *testing.T) {
	payload := `{
		"id": 100,
		"created_at": "2024-01-15T10:00:00Z",
		"updated_at": "2024-01-15T10:05:00Z",
		"commit_sha": "abc123",
		"review_request_id": 42,
		"status": "failed",
		"stack_id": 456,
		"technology": "terraform",
		"logs_stderr_count": 50,
		"logs_stdout_count": 200
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/stack_previews/org-uuid/100"
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

	preview, resp, err := client.Previews.Get(context.Background(), "org-uuid", 100)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if preview == nil {
		t.Fatal("expected non-nil preview")
		return
	}
	if preview.ID != 100 {
		t.Errorf("unexpected id: got %d, want 100", preview.ID)
	}
	if preview.Status != "failed" {
		t.Errorf("unexpected status: got %s", preview.Status)
	}
	if preview.LogsStderrCount != 50 {
		t.Errorf("unexpected logs_stderr_count: got %d, want 50", preview.LogsStderrCount)
	}
}

func TestPreviewsGet_Validation(t *testing.T) {
	c, err := NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tests := []struct {
		name           string
		orgUUID        string
		stackPreviewID int
		wantError      string
	}{
		{"empty org UUID", "", 100, "organization UUID is required"},
		{"zero preview ID", "org-uuid", 0, "stack preview ID must be positive"},
		{"negative preview ID", "org-uuid", -1, "stack preview ID must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.Previews.Get(context.Background(), tt.orgUUID, tt.stackPreviewID)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantError {
				t.Errorf("got error %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestPreviewsGetLogs_ParsesResponse(t *testing.T) {
	payload := `{
		"stack_preview_log_lines": [
			{
				"log_line": 1,
				"timestamp": "2024-01-15T10:00:00Z",
				"channel": "stderr",
				"message": "Error: Provider authentication failed"
			},
			{
				"log_line": 2,
				"timestamp": "2024-01-15T10:00:01Z",
				"channel": "stderr",
				"message": "AWS_ACCESS_KEY_ID not set"
			}
		],
		"paginated_result": {
			"page": 1,
			"per_page": 100,
			"total": 2
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/stack_previews/org-uuid/100/logs"
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

	logs, resp, err := client.Previews.GetLogs(context.Background(), "org-uuid", 100, nil)
	if err != nil {
		t.Fatalf("GetLogs error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if len(logs.StackPreviewLogLines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(logs.StackPreviewLogLines))
	}

	log := logs.StackPreviewLogLines[0]
	if log.Channel != "stderr" {
		t.Errorf("unexpected channel: got %s", log.Channel)
	}
	if log.Message != "Error: Provider authentication failed" {
		t.Errorf("unexpected message: got %s", log.Message)
	}
}

func TestPreviewsGetLogs_WithOptions(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("channel") != "stderr" {
			t.Errorf("expected channel=stderr, got %s", query.Get("channel"))
		}
		if query.Get("page") != "2" {
			t.Errorf("expected page=2, got %s", query.Get("page"))
		}

		payload := `{"stack_preview_log_lines":[],"paginated_result":{"page":2,"per_page":100,"total":0}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	opts := &PreviewLogsOptions{
		ListOptions: ListOptions{
			Page:    2,
			PerPage: 100,
		},
		Channel: "stderr",
	}

	_, _, err := client.Previews.GetLogs(context.Background(), "org-uuid", 100, opts)
	if err != nil {
		t.Fatalf("GetLogs error: %v", err)
	}
}

func TestPreviewsExplainErrors_ParsesResponse(t *testing.T) {
	payload := `{
		"summary": {
			"contents": [
				"Provider authentication failed",
				"AWS credentials are not configured",
				"Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY"
			],
			"created_at": "2024-01-15T10:05:00Z"
		}
	}`

	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/stack_previews/org-uuid/100/ai/error_logs_explanation"
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

	summary, resp, err := client.Previews.ExplainErrors(context.Background(), "org-uuid", 100, false)
	if err != nil {
		t.Fatalf("ExplainErrors error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
		return
	}
	if len(summary.Summary.Contents) != 3 {
		t.Fatalf("expected 3 summary lines, got %d", len(summary.Summary.Contents))
	}
	if summary.Summary.Contents[0] != "Provider authentication failed" {
		t.Errorf("unexpected summary: got %s", summary.Summary.Contents[0])
	}
}

func TestPreviewsExplainErrors_WithForce(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("force") != "true" {
			t.Errorf("expected force=true, got %s", query.Get("force"))
		}

		payload := `{"summary":{"contents":["test"],"created_at":"2024-01-15T10:05:00Z"}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, werr := w.Write([]byte(payload)); werr != nil {
			panic(werr)
		}
	})
	defer cleanup()

	_, _, err := client.Previews.ExplainErrors(context.Background(), "org-uuid", 100, true)
	if err != nil {
		t.Fatalf("ExplainErrors error: %v", err)
	}
}

func TestPreviewsGetLogs_Validation(t *testing.T) {
	c, err := NewClientWithAPIKey("key")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	tests := []struct {
		name           string
		orgUUID        string
		stackPreviewID int
		wantError      string
	}{
		{"empty org UUID", "", 100, "organization UUID is required"},
		{"zero preview ID", "org-uuid", 0, "stack preview ID must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.Previews.GetLogs(context.Background(), tt.orgUUID, tt.stackPreviewID, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantError {
				t.Errorf("got error %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestPreviewsGet_RespectsContextCancellation(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := client.Previews.Get(ctx, "org-uuid", 100)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestPreviewsGetLogs_RespectsContextTimeout(t *testing.T) {
	client, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err := client.Previews.GetLogs(ctx, "org-uuid", 100, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
