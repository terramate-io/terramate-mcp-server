package terramate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/terramate-io/terramate-mcp-server/internal/version"
)

const (
	defaultTimeout = 30 * time.Second
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	// retryCountKey is used to track the number of 401 retries in a request chain
	retryCountKey contextKey = "retry_count"
	maxRetries    int        = 1 // Maximum number of 401 retries per request
)

// Client is the main Terramate Cloud API client
type Client struct {
	// HTTP client used for requests
	httpClient *http.Client

	// Base URL for API requests
	baseURL *url.URL

	// Credential for authentication (JWT token or API key)
	credential Credential

	// User agent for requests
	userAgent string

	// Services
	Memberships    *MembershipsService
	Stacks         *StacksService
	Drifts         *DriftsService
	ReviewRequests *ReviewRequestsService
	Deployments    *DeploymentsService
	Previews       *PreviewsService
}

// ClientOption is a functional option for configuring the Client
type ClientOption func(*Client) error

// NewClient creates a new Terramate Cloud API client with the given credential
func NewClient(credential Credential, opts ...ClientOption) (*Client, error) {
	if credential == nil {
		return nil, fmt.Errorf("credential is required")
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
		baseURL:    baseURL,
		credential: credential,
		userAgent:  version.UserAgent(),
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
	client.ReviewRequests = &ReviewRequestsService{client: client}
	client.Deployments = &DeploymentsService{client: client}
	client.Previews = &PreviewsService{client: client}

	return client, nil
}

// NewClientWithAPIKey creates a new Terramate Cloud API client with an API key
// This is a convenience function for backward compatibility with API key authentication
func NewClientWithAPIKey(apiKey string, opts ...ClientOption) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	return NewClient(NewAPIKeyCredential(apiKey), opts...)
}

// NewClientWithJWT creates a new Terramate Cloud API client with a JWT token
// This is a convenience function for JWT token authentication
func NewClientWithJWT(jwtToken string, opts ...ClientOption) (*Client, error) {
	if jwtToken == "" {
		return nil, fmt.Errorf("JWT token is required")
	}

	credential, err := NewJWTCredential(jwtToken, "")
	if err != nil {
		return nil, fmt.Errorf("invalid JWT token: %w", err)
	}

	return NewClient(credential, opts...)
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
			base = "https://us.api.terramate.io"
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

	// Ensure GetBody is set for all body types to support request cloning/retry.
	// Go's http package only sets GetBody automatically for certain types like
	// *bytes.Buffer, *bytes.Reader, *strings.Reader. For custom io.Reader types,
	// we need to read the body into a buffer to enable cloning.
	var bodyReader io.Reader = body
	if body != nil {
		// Check if body is a type that Go's http package recognizes and sets GetBody for.
		// Known types: *bytes.Buffer, *bytes.Reader, *strings.Reader
		// For any other type, we buffer it to ensure GetBody is set.
		switch body.(type) {
		case *bytes.Buffer, *bytes.Reader:
			// These types automatically get GetBody set by Go's http package
			bodyReader = body
		default:
			// Check for *strings.Reader (can't include in switch due to package visibility)
			// For *strings.Reader and other custom io.Reader types, buffer to enable cloning
			if _, ok := body.(*strings.Reader); ok {
				// *strings.Reader also gets GetBody set automatically, use as-is
				bodyReader = body
			} else {
				// For custom io.Reader types, read into buffer to enable cloning
				bodyBytes, readErr := io.ReadAll(body)
				if readErr != nil {
					return nil, fmt.Errorf("failed to read request body: %w", readErr)
				}
				bodyReader = bytes.NewReader(bodyBytes)
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	const contentTypeJSON = "application/json"
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)

	// Apply credentials (JWT Bearer token or API Key Basic Auth)
	if err := c.credential.ApplyCredentials(req); err != nil {
		return nil, fmt.Errorf("failed to apply credentials: %w", err)
	}

	return req, nil
}

// do executes an HTTP request and handles the response.
// If the request fails with 401 Unauthorized and the client uses JWT authentication,
// it attempts to refresh the token and retry the request once.
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

	// Handle 401 Unauthorized - attempt token refresh if using JWT
	if resp.StatusCode == http.StatusUnauthorized {
		if refreshableCred, ok := c.credential.(RefreshableCredential); ok {
			// Check retry count to prevent unbounded recursion
			retryCount := 0
			if count, ok := req.Context().Value(retryCountKey).(int); ok {
				retryCount = count
			}
			if retryCount >= maxRetries {
				// Already retried once, don't retry again
				return response, parseAPIError(resp, body)
			}

			// Try to refresh the token
			if refreshErr := refreshableCred.Refresh(req.Context()); refreshErr == nil {
				// Token refreshed successfully - retry the request
				// Clone the request to avoid reusing the body
				retryReq, cloneErr := cloneRequest(req)
				if cloneErr == nil {
					// Apply the new credentials
					if applyErr := c.credential.ApplyCredentials(retryReq); applyErr == nil {
						// Increment retry count in context to prevent infinite recursion
						retryCtx := context.WithValue(retryReq.Context(), retryCountKey, retryCount+1)
						retryReq = retryReq.WithContext(retryCtx)
						// Recursively call do() for the retry (will not recurse again due to retry count check)
						return c.do(retryReq, v)
					}
				}
			}
		}
	}

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

// cloneRequest creates a clone of an HTTP request for retry purposes.
// This is necessary because http.Request.Body can only be read once.
func cloneRequest(req *http.Request) (*http.Request, error) {
	clonedReq := req.Clone(req.Context())

	// If the request had a body, we need to handle it specially
	if req.Body != nil {
		if req.GetBody != nil {
			// Use GetBody to get a fresh copy of the body
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to get request body: %w", err)
			}
			clonedReq.Body = body
		} else {
			// GetBody is nil - this should not happen if newRequest is working correctly,
			// but handle it defensively to prevent silent failures.
			// The body has already been consumed by the first request, so we cannot clone it.
			return nil, fmt.Errorf("cannot clone request: body GetBody is nil (body may have been consumed)")
		}
	}

	return clonedReq, nil
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
	// Default to generic error message to avoid leaking sensitive data
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Message:    fmt.Sprintf("API request failed with status %d", resp.StatusCode),
	}

	// Try to parse JSON error response safely
	if isJSONContentType(resp.Header.Get("Content-Type")) {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			// Only use parsed error fields, never raw body
			apiErr.Message = errResp.Error
			if apiErr.Message == "" {
				apiErr.Message = fmt.Sprintf("API request failed with status %d", resp.StatusCode)
			}
			apiErr.Details = errResp.Details
		}
	}

	// For 401 Unauthorized, provide helpful guidance
	if resp.StatusCode == http.StatusUnauthorized {
		apiErr.Message = fmt.Sprintf(
			"Authentication failed: %s\n\n"+
				"Your credentials may be invalid or expired.\n"+
				"To fix this:\n"+
				"  1. Run 'terramate cloud login' to refresh your JWT credentials\n"+
				"  2. Or provide a valid API key with --api-key flag\n"+
				"  3. Or set TERRAMATE_API_KEY environment variable",
			apiErr.Message,
		)
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

// Query builder helper functions

// addPagination adds pagination parameters to a query
func addPagination(query url.Values, page, perPage int) {
	if page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	if perPage > 0 {
		query.Set("per_page", strconv.Itoa(perPage))
	}
}

// addStringSlice adds a comma-separated string slice to a query
func addStringSlice(query url.Values, key string, values []string) {
	if len(values) > 0 {
		query.Set(key, strings.Join(values, ","))
	}
}

// addIntSlice adds a comma-separated int slice to a query
func addIntSlice(query url.Values, key string, values []int) {
	if len(values) > 0 {
		strValues := make([]string, len(values))
		for i, v := range values {
			strValues[i] = strconv.Itoa(v)
		}
		query.Set(key, strings.Join(strValues, ","))
	}
}

// addBoolSlice adds a comma-separated bool slice to a query
func addBoolSlice(query url.Values, key string, values []bool) {
	if len(values) > 0 {
		strValues := make([]string, len(values))
		for i, v := range values {
			strValues[i] = strconv.FormatBool(v)
		}
		query.Set(key, strings.Join(strValues, ","))
	}
}

// addBoolPtr adds a boolean pointer to a query if non-nil
func addBoolPtr(query url.Values, key string, value *bool) {
	if value != nil {
		query.Set(key, strconv.FormatBool(*value))
	}
}

// addString adds a string to a query if non-empty
func addString(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

// addTimePtr adds a timestamp to a query if non-nil
func addTimePtr(query url.Values, key string, value *time.Time) {
	if value != nil {
		query.Set(key, value.Format("2006-01-02T15:04:05Z07:00"))
	}
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
