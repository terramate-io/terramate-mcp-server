package terramate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	if opts.Page > 0 {
		query.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(opts.PerPage))
	}
}

// addFilterParams adds filter query parameters
func (opts *StacksListOptions) addFilterParams(query url.Values) {
	if len(opts.Repository) > 0 {
		query.Set("repository", strings.Join(opts.Repository, ","))
	}
	if len(opts.Target) > 0 {
		query.Set("target", strings.Join(opts.Target, ","))
	}
	if len(opts.Status) > 0 {
		query.Set("status", strings.Join(opts.Status, ","))
	}
	if len(opts.DeploymentStatus) > 0 {
		query.Set("deployment_status", strings.Join(opts.DeploymentStatus, ","))
	}
	if len(opts.DriftStatus) > 0 {
		query.Set("drift_status", strings.Join(opts.DriftStatus, ","))
	}
	if opts.Draft != nil {
		query.Set("draft", strconv.FormatBool(*opts.Draft))
	}
	if len(opts.IsArchived) > 0 {
		archived := make([]string, len(opts.IsArchived))
		for i, v := range opts.IsArchived {
			archived[i] = strconv.FormatBool(v)
		}
		query.Set("is_archived", strings.Join(archived, ","))
	}
	if opts.Search != "" {
		query.Set("search", opts.Search)
	}
	if opts.MetaID != "" {
		query.Set("meta_id", opts.MetaID)
	}
	if opts.DeploymentUUID != "" {
		query.Set("deployment_uuid", opts.DeploymentUUID)
	}
	if len(opts.PolicySeverity) > 0 {
		query.Set("policy_severity", strings.Join(opts.PolicySeverity, ","))
	}
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
