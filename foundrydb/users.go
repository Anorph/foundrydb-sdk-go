package foundrydb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListUsers returns all database users defined on the given managed service.
func (c *Client) ListUsers(ctx context.Context, serviceID string) ([]DatabaseUser, error) {
	path := "/managed-services/" + serviceID + "/database-users"
	resp, err := c.do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	data, err := checkResponse(resp)
	if err != nil {
		return nil, err
	}
	var result ListUsersResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("foundrydb: decode ListUsers response: %w", err)
	}
	return result.Users, nil
}

// RevealPassword fetches the full connection credentials (including the plaintext password)
// for the given database user on the given managed service.
func (c *Client) RevealPassword(ctx context.Context, serviceID, username string) (*RevealPasswordResponse, error) {
	path := fmt.Sprintf("/managed-services/%s/database-users/%s/reveal-password", serviceID, username)
	resp, err := c.do(ctx, http.MethodPost, path, nil, "")
	if err != nil {
		return nil, err
	}
	data, err := checkResponse(resp)
	if err != nil {
		return nil, err
	}
	var creds RevealPasswordResponse
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("foundrydb: decode RevealPassword response: %w", err)
	}
	return &creds, nil
}
