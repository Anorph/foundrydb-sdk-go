package foundrydb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListOrganizations_Success(t *testing.T) {
	want := []Organization{
		{ID: "org-1", Name: "Acme Corp", Slug: "acme", IsPersonal: false, Role: "admin"},
		{ID: "org-2", Name: "Personal", Slug: "personal", IsPersonal: true, Role: "owner"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/organizations" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListOrganizationsResponse{Organizations: want})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.ListOrganizations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 orgs, got %d", len(got))
	}
	if got[0].ID != "org-1" {
		t.Errorf("expected org-1, got %s", got[0].ID)
	}
	if got[1].IsPersonal != true {
		t.Errorf("expected IsPersonal=true for org-2")
	}
}

func TestListOrganizations_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ListOrganizationsResponse{Organizations: []Organization{}})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.ListOrganizations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d", len(got))
	}
}

func TestListOrganizations_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListOrganizations(context.Background())
	if !IsUnauthorized(err) {
		t.Errorf("expected IsUnauthorized, got %v", err)
	}
}

func TestListOrganizations_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListOrganizations(context.Background())
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestGetOrganization_Found(t *testing.T) {
	want := Organization{ID: "org-abc", Name: "My Org", Slug: "my-org", Role: "member"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/organizations/org-abc" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.GetOrganization(context.Background(), "org-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected org, got nil")
	}
	if got.ID != "org-abc" {
		t.Errorf("expected org-abc, got %s", got.ID)
	}
	if got.Name != "My Org" {
		t.Errorf("expected My Org, got %s", got.Name)
	}
}

func TestGetOrganization_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.GetOrganization(context.Background(), "missing-org")
	if err != nil {
		t.Fatalf("expected nil error for 404, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for 404, got %+v", got)
	}
}

func TestGetOrganization_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetOrganization(context.Background(), "org-1")
	if !IsForbidden(err) {
		t.Errorf("expected IsForbidden, got %v", err)
	}
}

func TestGetOrganization_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{bad}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetOrganization(context.Background(), "org-1")
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func networkErrorClientOrgs() *Client {
	return New(Config{
		APIURL:      "http://127.0.0.1:1",
		Username:    "admin",
		Password:    "pass",
		HTTPTimeout: 1 * time.Second,
	})
}

func TestListOrganizations_NetworkError(t *testing.T) {
	c := networkErrorClientOrgs()
	_, err := c.ListOrganizations(context.Background())
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestGetOrganization_NetworkError(t *testing.T) {
	c := networkErrorClientOrgs()
	_, err := c.GetOrganization(context.Background(), "org-1")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestListOrganizations_WithOrgIDHeader(t *testing.T) {
	var gotOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("X-Active-Org-ID")
		json.NewEncoder(w).Encode(ListOrganizationsResponse{Organizations: []Organization{}})
	}))
	defer srv.Close()

	c := newTestClientWithOrg(srv.URL, "scope-org")
	_, _ = c.ListOrganizations(context.Background())

	if gotOrg != "scope-org" {
		t.Errorf("expected X-Active-Org-ID=scope-org, got %q", gotOrg)
	}
}
