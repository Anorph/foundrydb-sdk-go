package foundrydb

import "fmt"

// APIError is returned when the FoundryDB API responds with a non-2xx status code.
type APIError struct {
	// StatusCode is the HTTP status code returned by the API.
	StatusCode int
	// Message is the human-readable error description.
	Message string
	// Body is the raw response body from the API.
	Body string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("foundrydb API error %d: %s", e.StatusCode, e.Message)
}

// IsNotFound returns true when the API returned a 404 Not Found response.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 404
	}
	return false
}

// IsUnauthorized returns true when the API returned a 401 Unauthorized response.
func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 401
	}
	return false
}

// IsForbidden returns true when the API returned a 403 Forbidden response.
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 403
	}
	return false
}
