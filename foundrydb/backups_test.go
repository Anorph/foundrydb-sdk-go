package foundrydb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func strPtr(s string) *string { return &s }

func networkErrorClientBackups() *Client {
	return New(Config{
		APIURL:      "http://127.0.0.1:1",
		Username:    "admin",
		Password:    "pass",
		HTTPTimeout: 1 * time.Second,
	})
}

func TestListBackups_NetworkError(t *testing.T) {
	c := networkErrorClientBackups()
	_, err := c.ListBackups(context.Background(), "svc-1")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestTriggerBackup_NetworkError(t *testing.T) {
	c := networkErrorClientBackups()
	_, err := c.TriggerBackup(context.Background(), "svc-1", CreateBackupRequest{BackupType: BackupTypeFull})
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestListBackups_Success(t *testing.T) {
	size := int64(1024 * 1024 * 500)
	completedAt := "2026-01-02T10:00:00Z"
	want := []Backup{
		{
			ID:          "bkp-1",
			ServiceID:   "svc-1",
			Status:      BackupStatusCompleted,
			BackupType:  BackupTypeFull,
			SizeBytes:   &size,
			CreatedAt:   "2026-01-02T09:00:00Z",
			CompletedAt: &completedAt,
		},
		{
			ID:         "bkp-2",
			ServiceID:  "svc-1",
			Status:     BackupStatusRunning,
			BackupType: BackupTypeIncremental,
			CreatedAt:  "2026-01-03T09:00:00Z",
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/managed-services/svc-1/backups" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ListBackupsResponse{Backups: want})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.ListBackups(context.Background(), "svc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(got))
	}
	if got[0].ID != "bkp-1" {
		t.Errorf("expected bkp-1, got %s", got[0].ID)
	}
	if got[0].Status != BackupStatusCompleted {
		t.Errorf("expected completed, got %s", got[0].Status)
	}
	if *got[0].SizeBytes != size {
		t.Errorf("expected size %d, got %d", size, *got[0].SizeBytes)
	}
}

func TestListBackups_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ListBackupsResponse{Backups: []Backup{}})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.ListBackups(context.Background(), "svc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d", len(got))
	}
}

func TestListBackups_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"service not found"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListBackups(context.Background(), "missing-svc")
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestListBackups_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{bad json`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.ListBackups(context.Background(), "svc-1")
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestTriggerBackup_Success(t *testing.T) {
	req := CreateBackupRequest{BackupType: BackupTypeFull}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/managed-services/svc-1/backups" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body CreateBackupRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		if body.BackupType != BackupTypeFull {
			t.Errorf("expected full backup type, got %s", body.BackupType)
		}
		w.WriteHeader(http.StatusCreated)
		// API returns backup_id (not id) in the trigger response envelope.
		w.Write([]byte(`{"backup_id":"new-bkp","status":"pending"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.TriggerBackup(context.Background(), "svc-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "new-bkp" {
		t.Errorf("expected new-bkp, got %s", got.ID)
	}
	if got.Status != BackupStatusPending {
		t.Errorf("expected pending, got %s", got.Status)
	}
}

func TestTriggerBackup_DefaultBackupType(t *testing.T) {
	// Empty backup type should still work (platform default)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		// API returns backup_id (not id) in the trigger response envelope.
		w.Write([]byte(`{"backup_id":"bkp-default","status":"pending"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.TriggerBackup(context.Background(), "svc-1", CreateBackupRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "bkp-default" {
		t.Errorf("expected bkp-default, got %s", got.ID)
	}
}

func TestTriggerBackup_IncrementalType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body CreateBackupRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.BackupType != BackupTypeIncremental {
			t.Errorf("expected incremental, got %s", body.BackupType)
		}
		// API returns backup_id (not id) in the trigger response envelope.
		w.Write([]byte(`{"backup_id":"bkp-incr","status":"pending"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.TriggerBackup(context.Background(), "svc-1", CreateBackupRequest{BackupType: BackupTypeIncremental})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTriggerBackup_PITRType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API returns backup_id (not id) in the trigger response envelope.
		w.Write([]byte(`{"backup_id":"bkp-pitr","status":"pending"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.TriggerBackup(context.Background(), "svc-1", CreateBackupRequest{BackupType: BackupTypePITR})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.BackupType != BackupTypePITR {
		t.Errorf("expected pitr, got %s", got.BackupType)
	}
}

func TestTriggerBackup_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"backup already in progress"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.TriggerBackup(context.Background(), "svc-1", CreateBackupRequest{BackupType: BackupTypeFull})
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
}

func TestTriggerBackup_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`oops`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.TriggerBackup(context.Background(), "svc-1", CreateBackupRequest{})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestListBackups_WithOrgIDHeader(t *testing.T) {
	var gotOrg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = r.Header.Get("X-Active-Org-ID")
		json.NewEncoder(w).Encode(ListBackupsResponse{Backups: []Backup{}})
	}))
	defer srv.Close()

	c := newTestClientWithOrg(srv.URL, "backup-org")
	_, _ = c.ListBackups(context.Background(), "svc-1")

	if gotOrg != "backup-org" {
		t.Errorf("expected X-Active-Org-ID=backup-org, got %q", gotOrg)
	}
}

func TestBackup_ErrorMessageField(t *testing.T) {
	errMsg := "disk full"
	want := Backup{
		ID:           "bkp-fail",
		ServiceID:    "svc-1",
		Status:       BackupStatusFailed,
		BackupType:   BackupTypeFull,
		ErrorMessage: &errMsg,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ListBackupsResponse{Backups: []Backup{want}})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	got, err := c.ListBackups(context.Background(), "svc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(got))
	}
	if got[0].Status != BackupStatusFailed {
		t.Errorf("expected failed status, got %s", got[0].Status)
	}
	if got[0].ErrorMessage == nil || *got[0].ErrorMessage != "disk full" {
		t.Errorf("expected error message 'disk full', got %v", got[0].ErrorMessage)
	}
}
