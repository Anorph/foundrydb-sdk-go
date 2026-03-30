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
	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("foundrydb: decode TriggerBackup response: %w", err)
	}
	return &backup, nil
}
