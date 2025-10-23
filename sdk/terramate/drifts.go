package terramate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DriftsService handles communication with the drifts related
// methods of the Terramate Cloud API
type DriftsService struct {
	client *Client
}

// buildQuery constructs URL query parameters from DriftsListOptions
func (opts *DriftsListOptions) buildQuery() url.Values {
	query := url.Values{}
	if opts == nil {
		return query
	}

	addPagination(query, opts.Page, opts.PerPage)
	addStringSlice(query, "drift_status", opts.DriftStatus)
	addString(query, "grouping_key", opts.GroupingKey)

	return query
}

// ListForStack retrieves all drift detection runs for a specific stack.
//
// GET /v1/stacks/{org_uuid}/{stack_id}/drifts
//
// This endpoint returns all drift detection runs for a specific stack.
//
// Access: All members of the organization with any role are allowed to query.
func (s *DriftsService) ListForStack(ctx context.Context, orgUUID string, stackID int, opts *DriftsListOptions) (*DriftsListResponse, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}
	if stackID <= 0 {
		return nil, nil, fmt.Errorf("stack ID must be positive")
	}

	path := fmt.Sprintf("/v1/stacks/%s/%d/drifts", orgUUID, stackID)

	// Build query parameters
	if opts != nil {
		query := opts.buildQuery()
		if len(query) > 0 {
			path = path + "?" + query.Encode()
		}
	}

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result DriftsListResponse
	resp, err := s.client.do(req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}

// Get retrieves detailed information for a specific drift.
//
// GET /v1/drifts/{org_uuid}/{stack_id}/{drift_id}
//
// This endpoint returns full drift details including drift_details with changeset_ascii
// (the terraform plan output) which can be up to 4MB in size.
//
// Access: All members of the organization with any role are allowed to query.
func (s *DriftsService) Get(ctx context.Context, orgUUID string, stackID, driftID int) (*Drift, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}
	if stackID <= 0 {
		return nil, nil, fmt.Errorf("stack ID must be positive")
	}
	if driftID <= 0 {
		return nil, nil, fmt.Errorf("drift ID must be positive")
	}

	path := fmt.Sprintf("/v1/drifts/%s/%d/%d", orgUUID, stackID, driftID)

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var drift Drift
	resp, err := s.client.do(req, &drift)
	if err != nil {
		return nil, resp, err
	}

	return &drift, resp, nil
}
