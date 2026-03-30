package foundrydb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListOrganizations returns all organizations the authenticated user belongs to.
func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	resp, err := c.do(ctx, http.MethodGet, "/organizations", nil, "")
	if err != nil {
		return nil, err
	}
	data, err := checkResponse(resp)
	if err != nil {
		return nil, err
	}
	var result ListOrganizationsResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("foundrydb: decode ListOrganizations response: %w", err)
	}
	return result.Organizations, nil
}

// GetOrganization returns the organization with the given UUID.
// Returns nil, nil when the organization does not exist (404).
func (c *Client) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	resp, err := c.do(ctx, http.MethodGet, "/organizations/"+id, nil, "")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, nil
	}
	data, err := checkResponse(resp)
	if err != nil {
		return nil, err
	}
	var org Organization
	if err := json.Unmarshal(data, &org); err != nil {
		return nil, fmt.Errorf("foundrydb: decode GetOrganization response: %w", err)
	}
	return &org, nil
}
