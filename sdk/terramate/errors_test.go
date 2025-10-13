package terramate

import (
	"net/http"
	"testing"
)

func TestAPIError_ErrorMessage(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Message:    "not found",
	}
	if err.Error() != "API error (status 404): not found" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}

	errWithDetails := &APIError{
		StatusCode: 400,
		Message:    "bad request",
		Details:    map[string]interface{}{"field": "value"},
	}
	expected := "API error (status 400): bad request - map[field:value]"
	if errWithDetails.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, errWithDetails.Error())
	}
}

func TestAPIError_IsNotFound(t *testing.T) {
	err := &APIError{StatusCode: http.StatusNotFound}
	if !err.IsNotFound() {
		t.Fatal("expected IsNotFound to be true")
	}
	err2 := &APIError{StatusCode: 400}
	if err2.IsNotFound() {
		t.Fatal("expected IsNotFound to be false")
	}
}

func TestAPIError_IsUnauthorized(t *testing.T) {
	err := &APIError{StatusCode: http.StatusUnauthorized}
	if !err.IsUnauthorized() {
		t.Fatal("expected IsUnauthorized to be true")
	}
}

func TestAPIError_IsForbidden(t *testing.T) {
	err := &APIError{StatusCode: http.StatusForbidden}
	if !err.IsForbidden() {
		t.Fatal("expected IsForbidden to be true")
	}
}

func TestAPIError_IsBadRequest(t *testing.T) {
	err := &APIError{StatusCode: http.StatusBadRequest}
	if !err.IsBadRequest() {
		t.Fatal("expected IsBadRequest to be true")
	}
}

func TestAPIError_IsServerError(t *testing.T) {
	err := &APIError{StatusCode: 500}
	if !err.IsServerError() {
		t.Fatal("expected IsServerError to be true for 500")
	}
	err2 := &APIError{StatusCode: 503}
	if !err2.IsServerError() {
		t.Fatal("expected IsServerError to be true for 503")
	}
	err3 := &APIError{StatusCode: 400}
	if err3.IsServerError() {
		t.Fatal("expected IsServerError to be false for 400")
	}
}

func TestAPIError_IsClientError(t *testing.T) {
	err := &APIError{StatusCode: 400}
	if !err.IsClientError() {
		t.Fatal("expected IsClientError to be true for 400")
	}
	err2 := &APIError{StatusCode: 500}
	if err2.IsClientError() {
		t.Fatal("expected IsClientError to be false for 500")
	}
}

func TestErrorResponse_String(t *testing.T) {
	er := &ErrorResponse{Error: "error"}
	if er.String() != "error" {
		t.Fatalf("unexpected string: %s", er.String())
	}

	erWithDetails := &ErrorResponse{
		Error:   "error",
		Details: map[string]interface{}{"field": "value"},
	}
	expected := "error: map[field:value]"
	if erWithDetails.String() != expected {
		t.Fatalf("expected %q, got %q", expected, erWithDetails.String())
	}
}
