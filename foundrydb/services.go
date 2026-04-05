package foundrydb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ListServices returns all managed database services visible to the authenticated user.
// When the client was created with an OrgID, only services belonging to that organization
// are returned.
func (c *Client) ListServices(ctx context.Context) ([]Service, error) {
	resp, err := c.do(ctx, http.MethodGet, "/managed-services", nil, "")
	if err != nil {
		return nil, err
	}
	data, err := checkResponse(resp)
	if err != nil {
		return nil, err
	}
	var result ListServicesResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("foundrydb: decode ListServices response: %w", err)
	}
	return result.Services, nil
}

// GetService returns the managed service with the given UUID.
// Returns nil, nil when the service does not exist (404).
func (c *Client) GetService(ctx context.Context, id string) (*Service, error) {
	resp, err := c.do(ctx, http.MethodGet, "/managed-services/"+id, nil, "")
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
	var svc Service
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("foundrydb: decode GetService response: %w", err)
	}
	return &svc, nil
}

// CreateService provisions a new managed database service and returns its initial state.
// The service will be in "provisioning" status; use WaitForRunning to block until
// the service is ready to accept connections.
func (c *Client) CreateService(ctx context.Context, req CreateServiceRequest) (*Service, error) {
	resp, err := c.do(ctx, http.MethodPost, "/managed-services", req, "")
	if err != nil {
		return nil, err
	}
	data, err := checkResponse(resp)
	if err != nil {
		return nil, err
	}
	var svc Service
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("foundrydb: decode CreateService response: %w", err)
	}
	return &svc, nil
}

// UpdateService applies the given patch to the managed service and returns the updated state.
func (c *Client) UpdateService(ctx context.Context, id string, req UpdateServiceRequest) (*Service, error) {
	resp, err := c.do(ctx, http.MethodPatch, "/managed-services/"+id, req, "")
	if err != nil {
		return nil, err
	}
	data, err := checkResponse(resp)
	if err != nil {
		return nil, err
	}
	var svc Service
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("foundrydb: decode UpdateService response: %w", err)
	}
	return &svc, nil
}

// DeleteService initiates deletion of the managed service with the given UUID.
// A 404 response is treated as success (idempotent).
func (c *Client) DeleteService(ctx context.Context, id string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/managed-services/"+id, nil, "")
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil
	}
	_, err = checkResponse(resp)
	return err
}

// ListPresets returns all available service presets for AI agent workloads.
func (c *Client) ListPresets(ctx context.Context) (json.RawMessage, error) {
	resp, err := c.do(ctx, http.MethodGet, "/managed-services/presets", nil, "")
	if err != nil {
		return nil, err
	}
	data, err := checkResponse(resp)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// WaitForRunning polls the service until it reaches "running" status or the timeout expires.
// Polling interval is 10 seconds. The context deadline (if any) takes precedence over timeout.
//
// Returns an error immediately when the service enters a terminal failure state ("failed", "error").
func (c *Client) WaitForRunning(ctx context.Context, id string, timeout time.Duration) (*Service, error) {
	deadline := time.Now().Add(timeout)
	for {
		svc, err := c.GetService(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("foundrydb: polling service %s: %w", id, err)
		}
		if svc == nil {
			return nil, fmt.Errorf("foundrydb: service %s not found while waiting for running status", id)
		}

		status := strings.ToLower(string(svc.Status))
		if status == "running" {
			return svc, nil
		}
		if status == "failed" || status == "error" {
			return nil, fmt.Errorf("foundrydb: service %s entered terminal status %q", id, svc.Status)
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("foundrydb: timed out after %s waiting for service %s to reach running status (current: %s)",
				timeout, id, svc.Status)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
}
