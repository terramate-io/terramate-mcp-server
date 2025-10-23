package terramate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/terramate-io/terramate-mcp-server/internal/version"
)

const (
	defaultTimeout = 30 * time.Second
)

// Client is the main Terramate Cloud API client
type Client struct {
	// HTTP client used for requests
	httpClient *http.Client

	// Base URL for API requests
	baseURL *url.URL

	// API key for authentication
	apiKey string

	// User agent for requests
	userAgent string

	// Services
	Memberships *MembershipsService
	Stacks      *StacksService
	Drifts      *DriftsService
}

// ClientOption is a functional option for configuring the Client
type ClientOption func(*Client) error

// NewClient creates a new Terramate Cloud API client
func NewClient(apiKey string, opts ...ClientOption) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Default base URL
	baseURL, err := url.Parse("https://api.terramate.io")
	if err != nil {
		return nil, fmt.Errorf("failed to parse default base URL: %w", err)
	}

	client := &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		baseURL:   baseURL,
		apiKey:    apiKey,
		userAgent: version.UserAgent(),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("failed to apply client option: %w", err)
		}
	}

	// Initialize services
	client.Memberships = &MembershipsService{client: client}
	client.Stacks = &StacksService{client: client}
	client.Drifts = &DriftsService{client: client}

	return client, nil
}

// WithBaseURL sets a custom base URL for the API
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		u, err := url.Parse(baseURL)
		if err != nil {
			return fmt.Errorf("invalid base URL: %w", err)
		}
		c.baseURL = u
		return nil
	}
}

// WithRegion configures base URL based on region shortcut ("us" or "eu").
// If WithBaseURL is also provided, it will override this region setting.
func WithRegion(region string) ClientOption {
	return func(c *Client) error {
		if region == "" {
			return nil
		}
		var base string
		switch region {
		case "us":
			base = "https://api.us.terramate.io"
		case "eu":
			base = "https://api.terramate.io"
		default:
			return fmt.Errorf("invalid region: %q", region)
		}
		u, err := url.Parse(base)
		if err != nil {
			return fmt.Errorf("invalid region base URL %q: %w", base, err)
		}
		c.baseURL = u
		return nil
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) error {
		if httpClient == nil {
			return fmt.Errorf("HTTP client cannot be nil")
		}
		c.httpClient = httpClient
		return nil
	}
}

// WithTimeout sets a custom timeout for requests
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) error {
		c.httpClient.Timeout = timeout
		return nil
	}
}

//nolint:unparam // method parameter will be used with different HTTP methods as SDK grows
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	// Build full URL
	u, err := c.baseURL.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL path: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	const contentTypeJSON = "application/json"
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)

	// API key authentication: Basic Auth with API key as username, empty password
	// @TODO: Needs to be updated in the future whenever we want to start supporting JWT tokens
	req.SetBasicAuth(c.apiKey, "")

	return req, nil
}

// do executes an HTTP request and handles the response
func (c *Client) do(req *http.Request, v interface{}) (*Response, error) {
	const maxBodyBytes = 10 << 20 // 10 MiB
	resp, err := c.executeRequestWithRetries(req, 3)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	response := &Response{HTTPResponse: resp, Body: body}

	if resp.StatusCode >= 400 {
		return response, parseAPIError(resp, body)
	}

	if resp.StatusCode == http.StatusNoContent || len(body) == 0 {
		return response, nil
	}

	if v != nil {
		if err := decodeJSONIfApplicable(resp, body, v); err != nil {
			return response, err
		}
	}

	return response, nil
}

func (c *Client) executeRequestWithRetries(req *http.Request, maxRetries int) (*http.Response, error) {
	isIdempotent := req.Method == http.MethodGet || req.Method == http.MethodHead || req.Method == http.MethodOptions
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if isIdempotent && attempt < maxRetries && req.Context().Err() == nil {
				if wait := backoffForAttempt(attempt); !sleepOrCtxDone(req.Context(), wait) {
					continue
				}
			}
			return nil, fmt.Errorf("request failed: %w", err)
		}
		if isIdempotent && shouldRetryStatus(resp.StatusCode) {
			if attempt < maxRetries {
				_ = resp.Body.Close()
				if wait := backoffForAttempt(attempt); sleepOrCtxDone(req.Context(), wait) {
					// Context was canceled during backoff
					return nil, req.Context().Err()
				}
				continue
			}
			// On final attempt with retryable status, return error
			_ = resp.Body.Close()
			return nil, fmt.Errorf("request failed with status %d after %d retries", resp.StatusCode, maxRetries)
		}
		return resp, nil
	}
	return nil, fmt.Errorf("exceeded retry attempts")
}

func shouldRetryStatus(code int) bool {
	return code == http.StatusTooManyRequests || (code >= 500 && code < 600)
}

func backoffForAttempt(attempt int) time.Duration {
	return time.Duration(100*(1<<attempt)) * time.Millisecond
}

func sleepOrCtxDone(ctx context.Context, d time.Duration) bool {
	select {
	case <-time.After(d):
		return false
	case <-ctx.Done():
		return true
	}
}

func parseAPIError(resp *http.Response, body []byte) error {
	apiErr := &APIError{StatusCode: resp.StatusCode, Message: string(body)}
	if isJSONContentType(resp.Header.Get("Content-Type")) {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			apiErr.Message = errResp.Error
			apiErr.Details = errResp.Details
		}
	}
	return apiErr
}

func decodeJSONIfApplicable(resp *http.Response, body []byte, v interface{}) error {
	if isJSONContentType(resp.Header.Get("Content-Type")) {
		if err := json.Unmarshal(body, v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

func isJSONContentType(ct string) bool {
	if ct == "" {
		return false
	}
	const contentTypeJSON = "application/json"
	return ct == contentTypeJSON || (len(ct) >= len(contentTypeJSON) && ct[:len(contentTypeJSON)] == contentTypeJSON)
}

// Response wraps the HTTP response
type Response struct {
	HTTPResponse *http.Response
	Body         []byte
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// String returns a string representation of the error response
func (e *ErrorResponse) String() string {
	if len(e.Details) == 0 {
		return e.Error
	}
	return fmt.Sprintf("%s: %v", e.Error, e.Details)
}
