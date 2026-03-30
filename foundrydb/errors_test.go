package foundrydb

import (
	"errors"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 404, Message: "not found", Body: `{"error":"not found"}`}
	want := "foundrydb API error 404: not found"
	if got := e.Error(); got != want {
		t.Errorf("APIError.Error() = %q, want %q", got, want)
	}
}

func TestAPIError_ErrorEmptyMessage(t *testing.T) {
	e := &APIError{StatusCode: 500, Message: "", Body: ""}
	want := "foundrydb API error 500: "
	if got := e.Error(); got != want {
		t.Errorf("APIError.Error() = %q, want %q", got, want)
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"404 APIError", &APIError{StatusCode: 404}, true},
		{"403 APIError", &APIError{StatusCode: 403}, false},
		{"500 APIError", &APIError{StatusCode: 500}, false},
		{"non-APIError", errors.New("some error"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"401 APIError", &APIError{StatusCode: 401}, true},
		{"403 APIError", &APIError{StatusCode: 403}, false},
		{"500 APIError", &APIError{StatusCode: 500}, false},
		{"non-APIError", errors.New("some error"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUnauthorized(tt.err); got != tt.want {
				t.Errorf("IsUnauthorized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsForbidden(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"403 APIError", &APIError{StatusCode: 403}, true},
		{"401 APIError", &APIError{StatusCode: 401}, false},
		{"500 APIError", &APIError{StatusCode: 500}, false},
		{"non-APIError", errors.New("some error"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsForbidden(tt.err); got != tt.want {
				t.Errorf("IsForbidden() = %v, want %v", got, tt.want)
			}
		})
	}
}
