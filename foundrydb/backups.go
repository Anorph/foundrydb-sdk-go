package foundrydb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListBackups returns all backup records for the given managed service, newest first.
func (c *Client) ListBackups(ctx context.Context, serviceID string) ([]Backup, error) {
	path := "/managed-services/" + serviceID + "/backups"
	resp, err := c.do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	data, err := checkResponse(resp)
	if err != nil {
		return nil, err
	}
	var result ListBackupsResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("foundrydb: decode ListBackups response: %w", err)
	}
	return result.Backups, nil
}

// triggerBackupResponse is the raw envelope returned by POST /managed-services/{id}/backups.
// The API returns backup_id instead of id, so we normalize it into a Backup.
type triggerBackupResponse struct {
	BackupID string       `json:"backup_id"`
	Status   BackupStatus `json:"status"`
	Message  string       `json:"message"`
	TaskID   string       `json:"task_id"`
}

// TriggerBackup requests an on-demand backup for the given managed service.
// Use req.BackupType to select "full", "incremental", or "pitr"; leave empty for the
// platform default (full).
func (c *Client) TriggerBackup(ctx context.Context, serviceID string, req CreateBackupRequest) (*Backup, error) {
	path := "/managed-services/" + serviceID + "/backups"
	resp, err := c.do(ctx, http.MethodPost, path, req, "")
	if err != nil {
		return nil, err
	}
	data, err := checkResponse(resp)
	if err != nil {
		return nil, err
	}
	var raw triggerBackupResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("foundrydb: decode TriggerBackup response: %w", err)
	}
	return &Backup{
		ID:         raw.BackupID,
		ServiceID:  serviceID,
		Status:     raw.Status,
		BackupType: req.BackupType,
	}, nil
}
