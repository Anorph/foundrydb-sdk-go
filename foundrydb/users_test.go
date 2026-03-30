package foundrydb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListUsers_Success(t *testing.T) {
	want := []DatabaseUser{
		{Username: "app_user", Roles: []string{"readwrite"}, CreatedAt: "2026-01-01T00:00:00Z"},
		{Username: "admin", Roles: []string{"superuser"}, CreatedAt: "2026-01-01T00:00:00Z"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/managed-services/svc-1/database-users" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListUsersResponse{Users: want})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.ListUsers(context.Background(), "svc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 users, got %d", len(got))
	}
	if got[0].Username != "app_user" {
		t.Errorf("expected app_user, got %s", got[0].Username)
	}
}

func TestListUsers_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ListUsersResponse{Users: []DatabaseUser{}})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.ListUsers(context.Background(), "svc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d", len(got))
	}
}

func TestListUsers_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"service not found"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListUsers(context.Background(), "missing-svc")
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestListUsers_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListUsers(context.Background(), "svc-1")
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestRevealPassword_Success(t *testing.T) {
	want := RevealPasswordResponse{
		Username:         "app_user",
		Password:         "s3cr3t",
		Host:             "pg.example.com",
		Port:             5432,
		Database:         "defaultdb",
		ConnectionString: "postgresql://app_user:s3cr3t@pg.example.com:5432/defaultdb",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/managed-services/svc-1/database-users/app_user/reveal-password" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.RevealPassword(context.Background(), "svc-1", "app_user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Password != "s3cr3t" {
		t.Errorf("expected s3cr3t, got %s", got.Password)
	}
	if got.Port != 5432 {
		t.Errorf("expected port 5432, got %d", got.Port)
	}
	if got.ConnectionString == "" {
		t.Error("expected non-empty connection string")
	}
}

func TestRevealPassword_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"user not found"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.RevealPassword(context.Background(), "svc-1", "no-user")
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestRevealPassword_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.RevealPassword(context.Background(), "svc-1", "app_user")
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestRevealPassword_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.RevealPassword(context.Background(), "svc-1", "app_user")
	if !IsUnauthorized(err) {
		t.Errorf("expected IsUnauthorized, got %v", err)
	}
}

func networkErrorClientUsers() *Client {
	return New(Config{
		APIURL:      "http://127.0.0.1:1",
		Username:    "admin",
		Password:    "pass",
		HTTPTimeout: 1 * time.Second,
	})
}

func TestListUsers_NetworkError(t *testing.T) {
	c := networkErrorClientUsers()
	_, err := c.ListUsers(context.Background(), "svc-1")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestRevealPassword_NetworkError(t *testing.T) {
	c := networkErrorClientUsers()
	_, err := c.RevealPassword(context.Background(), "svc-1", "app_user")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestListUsers_WithOrgIDHeader(t *testing.T) {
	var gotOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("X-Active-Org-ID")
		json.NewEncoder(w).Encode(ListUsersResponse{Users: []DatabaseUser{}})
	}))
	defer srv.Close()

	c := newTestClientWithOrg(srv.URL, "user-org")
	_, _ = c.ListUsers(context.Background(), "svc-1")

	if gotOrg != "user-org" {
		t.Errorf("expected X-Active-Org-ID=user-org, got %q", gotOrg)
	}
}
