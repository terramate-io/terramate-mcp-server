package terramate

import (
	"fmt"
	"net/http"
)

const (
	// ErrAuthenticationFailed is the error message returned when API authentication fails
	ErrAuthenticationFailed = "Authentication failed: Invalid API key"
)

// APIError represents an error returned by the Terramate Cloud API
type APIError struct {
	StatusCode int
	Message    string
	Details    map[string]interface{}
}

// Error implements the error interface
func (e *APIError) Error() string {
	if len(e.Details) == 0 {
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (status %d): %s - %v", e.StatusCode, e.Message, e.Details)
}

// IsNotFound returns true if the error is a 404 Not Found error
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsUnauthorized returns true if the error is a 401 Unauthorized error
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsForbidden returns true if the error is a 403 Forbidden error
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsBadRequest returns true if the error is a 400 Bad Request error
func (e *APIError) IsBadRequest() bool {
	return e.StatusCode == http.StatusBadRequest
}

// IsServerError returns true if the error is a 5xx server error
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// IsClientError returns true if the error is a 4xx client error
func (e *APIError) IsClientError() bool {
	return e.StatusCode >= 400 && e.StatusCode < 500
}
