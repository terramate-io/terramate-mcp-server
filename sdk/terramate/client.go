package terramate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	userAgent      = "terramate-mcp-server/1.0.0"
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
		userAgent: userAgent,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("failed to apply client option: %w", err)
		}
	}

	// Initialize services
	client.Memberships = &MembershipsService{client: client}

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

// newRequest creates a new HTTP request with common headers
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
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	return req, nil
}

// do executes an HTTP request and handles the response
func (c *Client) do(req *http.Request, v interface{}) (*Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	response := &Response{
		HTTPResponse: resp,
		Body:         body,
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}

		// Try to parse error response
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			apiErr.Message = errResp.Error
			apiErr.Details = errResp.Details
		}

		return response, apiErr
	}

	// Decode response if v is provided
	if v != nil && len(body) > 0 {
		if err := json.Unmarshal(body, v); err != nil {
			return response, fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return response, nil
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

// buildURL builds a URL with query parameters
func buildURL(base string, params map[string]string) string {
	if len(params) == 0 {
		return base
	}

	values := url.Values{}
	for key, value := range params {
		if value != "" {
			values.Set(key, value)
		}
	}

	if query := values.Encode(); query != "" {
		return base + "?" + query
	}

	return base
}

// formatJSON formats a JSON payload for requests
func formatJSON(v interface{}) (io.Reader, error) {
	if v == nil {
		return nil, nil
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return strings.NewReader(string(b)), nil
}
