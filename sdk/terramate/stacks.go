package terramate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// StacksService handles communication with the stacks related
// methods of the Terramate Cloud API
type StacksService struct {
	client *Client
}

// buildQuery constructs URL query parameters from StacksListOptions
func (opts *StacksListOptions) buildQuery() url.Values {
	query := url.Values{}
	if opts == nil {
		return query
	}

	opts.addPaginationParams(query)
	opts.addFilterParams(query)
	opts.addArrayParams(query)

	return query
}

// addPaginationParams adds pagination query parameters
func (opts *StacksListOptions) addPaginationParams(query url.Values) {
	addPagination(query, opts.Page, opts.PerPage)
}

// addFilterParams adds filter query parameters
func (opts *StacksListOptions) addFilterParams(query url.Values) {
	addStringSlice(query, "repository", opts.Repository)
	addStringSlice(query, "target", opts.Target)
	addStringSlice(query, "status", opts.Status)
	addStringSlice(query, "deployment_status", opts.DeploymentStatus)
	addStringSlice(query, "drift_status", opts.DriftStatus)
	addBoolPtr(query, "draft", opts.Draft)
	addBoolSlice(query, "is_archived", opts.IsArchived)
	addString(query, "search", opts.Search)
	addString(query, "meta_id", opts.MetaID)
	addString(query, "deployment_uuid", opts.DeploymentUUID)
	addStringSlice(query, "policy_severity", opts.PolicySeverity)
}

// addArrayParams adds array query parameters that use query.Add
func (opts *StacksListOptions) addArrayParams(query url.Values) {
	for _, tag := range opts.MetaTag {
		query.Add("meta_tag", tag)
	}
	for _, sort := range opts.Sort {
		query.Add("sort", sort)
	}
}

// List retrieves all stacks for an organization.
//
// GET /v1/stacks/{org_uuid}
//
// This endpoint returns stacks matching the provided filters.
// Stacks that are archived are not returned (use is_archived filter for archived stacks).
//
// Access: Members of the organization with any role are allowed to query.
func (s *StacksService) List(ctx context.Context, orgUUID string, opts *StacksListOptions) (*StacksListResponse, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}

	path := fmt.Sprintf("/v1/stacks/%s", orgUUID)

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

	var result StacksListResponse
	resp, err := s.client.do(req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}

// Get retrieves a specific stack by ID.
//
// GET /v1/stacks/{org_uuid}/{stack_id}
//
// This endpoint returns details for a specific stack.
//
// Access: All members of the organization with any role are allowed to query.
func (s *StacksService) Get(ctx context.Context, orgUUID string, stackID int) (*Stack, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}
	if stackID <= 0 {
		return nil, nil, fmt.Errorf("stack ID must be positive")
	}

	path := fmt.Sprintf("/v1/stacks/%s/%d", orgUUID, stackID)

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var stack Stack
	resp, err := s.client.do(req, &stack)
	if err != nil {
		return nil, resp, err
	}

	return &stack, resp, nil
}
