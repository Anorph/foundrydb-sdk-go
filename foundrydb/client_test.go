package foundrydb

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// unmarshalableBody is a type that cannot be JSON-marshaled.
type unmarshalableBody struct {
	Ch chan int
}

// errReader is an io.Reader that always returns an error.
type errReader struct{}

func (e *errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

// newTestClient creates a Client pointed at the given httptest server URL.
func newTestClient(serverURL string) *Client {
	return New(Config{
		APIURL:   serverURL,
		Username: "admin",
		Password: "pass",
	})
}

// newTestClientWithToken creates a Client using Bearer token auth.
func newTestClientWithToken(serverURL, token string) *Client {
	return New(Config{
		APIURL: serverURL,
		Token:  token,
	})
}

// newTestClientWithOrg creates a Client with an OrgID set.
func newTestClientWithOrg(serverURL, orgID string) *Client {
	return New(Config{
		APIURL:   serverURL,
		Username: "admin",
		Password: "pass",
		OrgID:    orgID,
	})
}

// --- New() / Config tests ---

func TestNew_DefaultAPIURL(t *testing.T) {
	c := New(Config{Username: "admin", Password: "pass"})
	if c.cfg.APIURL != defaultAPIURL {
		t.Errorf("expected default URL %q, got %q", defaultAPIURL, c.cfg.APIURL)
	}
}

func TestNew_TrailingSlashStripped(t *testing.T) {
	c := New(Config{APIURL: "https://example.com/", Username: "admin", Password: "pass"})
	if strings.HasSuffix(c.cfg.APIURL, "/") {
		t.Errorf("APIURL should not have trailing slash, got %q", c.cfg.APIURL)
	}
}

func TestNew_DefaultTimeout(t *testing.T) {
	c := New(Config{Username: "admin", Password: "pass"})
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", c.httpClient.Timeout)
	}
}

func TestNew_CustomTimeout(t *testing.T) {
	c := New(Config{Username: "admin", Password: "pass", HTTPTimeout: 5 * time.Second})
	if c.httpClient.Timeout != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v", c.httpClient.Timeout)
	}
}

// --- do() auth header tests ---

func TestDo_BasicAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"services":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, _ = c.ListServices(context.Background())

	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Errorf("expected Basic auth, got %q", gotAuth)
	}
}

func TestDo_BearerTokenAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"services":[]}`))
	}))
	defer srv.Close()

	c := newTestClientWithToken(srv.URL, "mytoken")
	_, _ = c.ListServices(context.Background())

	if gotAuth != "Bearer mytoken" {
		t.Errorf("expected Bearer auth, got %q", gotAuth)
	}
}

func TestDo_OrgIDHeader(t *testing.T) {
	var gotOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("X-Active-Org-ID")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"services":[]}`))
	}))
	defer srv.Close()

	c := newTestClientWithOrg(srv.URL, "org-123")
	_, _ = c.ListServices(context.Background())

	if gotOrg != "org-123" {
		t.Errorf("expected X-Active-Org-ID = org-123, got %q", gotOrg)
	}
}

func TestDo_NoOrgIDHeaderWhenEmpty(t *testing.T) {
	var gotOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("X-Active-Org-ID")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"services":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, _ = c.ListServices(context.Background())

	if gotOrg != "" {
		t.Errorf("expected no X-Active-Org-ID, got %q", gotOrg)
	}
}

func TestDo_ContentTypeSetForBody(t *testing.T) {
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"1","name":"svc","status":"running"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, _ = c.CreateService(context.Background(), CreateServiceRequest{Name: "svc", DatabaseType: PostgreSQL, PlanName: "tier-2"})

	if gotCT != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", gotCT)
	}
}

func TestDo_AcceptHeader(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"services":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, _ = c.ListServices(context.Background())

	if gotAccept != "application/json" {
		t.Errorf("expected Accept: application/json, got %q", gotAccept)
	}
}

func TestDo_NetworkError(t *testing.T) {
	// Point client at a port that won't respond.
	c := New(Config{
		APIURL:      "http://127.0.0.1:1",
		Username:    "admin",
		Password:    "pass",
		HTTPTimeout: 1 * time.Second,
	})
	_, err := c.ListServices(context.Background())
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestDo_CancelledContext(t *testing.T) {
	// Server that blocks.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c := newTestClient(srv.URL)
	_, err := c.ListServices(ctx)
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
}

// --- checkResponse() status code tests ---

func TestCheckResponse_200OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"services":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	svcs, err := c.ListServices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svcs == nil {
		t.Error("expected non-nil slice")
	}
}

func TestCheckResponse_201Created(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"new-id","name":"svc","status":"provisioning"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	svc, err := c.CreateService(context.Background(), CreateServiceRequest{Name: "svc", DatabaseType: PostgreSQL, PlanName: "tier-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.ID != "new-id" {
		t.Errorf("expected ID new-id, got %q", svc.ID)
	}
}

func TestCheckResponse_400BadRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid request"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.CreateService(context.Background(), CreateServiceRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("expected 400, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid request" {
		t.Errorf("expected 'invalid request', got %q", apiErr.Message)
	}
}

func TestCheckResponse_401Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListServices(context.Background())
	if !IsUnauthorized(err) {
		t.Errorf("expected IsUnauthorized, got %v", err)
	}
}

func TestCheckResponse_403Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListServices(context.Background())
	if !IsForbidden(err) {
		t.Errorf("expected IsForbidden, got %v", err)
	}
}

func TestCheckResponse_500InternalServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListServices(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("expected 500, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "internal error" {
		t.Errorf("expected 'internal error', got %q", apiErr.Message)
	}
}

func TestCheckResponse_ErrorBodyTruncated(t *testing.T) {
	longBody := strings.Repeat("x", 300)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(longBody))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListServices(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	// Message should be truncated to 200 chars + "..."
	if len(apiErr.Message) > 210 {
		t.Errorf("expected truncated message, got length %d", len(apiErr.Message))
	}
	if !strings.HasSuffix(apiErr.Message, "...") {
		t.Errorf("expected message to end with '...', got %q", apiErr.Message)
	}
}

func TestCheckResponse_MalformedJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListServices(context.Background())
	if err == nil {
		t.Fatal("expected JSON decode error, got nil")
	}
}

func TestDo_InvalidMethod(t *testing.T) {
	// http.NewRequestWithContext returns an error for methods containing spaces/special chars
	c := newTestClient("http://localhost:1")
	// Method with a space is invalid per net/http
	_, err := c.do(context.Background(), "INVALID METHOD", "/test", nil, "")
	if err == nil {
		t.Fatal("expected create request error, got nil")
	}
	if !strings.Contains(err.Error(), "create request") {
		t.Errorf("expected 'create request' in error, got: %v", err)
	}
}

func TestDo_MarshalError(t *testing.T) {
	// unmarshalableBody cannot be JSON marshaled (channels are not serializable)
	c := newTestClient("http://localhost:1")
	_, err := c.do(context.Background(), http.MethodPost, "/test", unmarshalableBody{Ch: make(chan int)}, "")
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}
	if !strings.Contains(err.Error(), "marshal request") {
		t.Errorf("expected 'marshal request' in error, got: %v", err)
	}
}

func TestCheckResponse_ReadBodyError(t *testing.T) {
	// Construct a response with a broken body reader
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(&errReader{}),
	}
	_, err := checkResponse(resp)
	if err == nil {
		t.Fatal("expected read error, got nil")
	}
	if !strings.Contains(err.Error(), "read response body") {
		t.Errorf("expected 'read response body' in error, got: %v", err)
	}
}

func TestCheckResponse_ErrorWithNoJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("service unavailable"))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListServices(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 503 {
		t.Errorf("expected 503, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "service unavailable" {
		t.Errorf("expected plain text message, got %q", apiErr.Message)
	}
}
