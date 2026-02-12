package terramate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ResourcesService handles communication with the resources-related
// methods of the Terramate Cloud API.
type ResourcesService struct {
	client *Client
}

// buildQuery constructs URL query parameters from ResourcesListOptions.
func (opts *ResourcesListOptions) buildQuery() url.Values {
	query := url.Values{}
	if opts == nil {
		return query
	}

	addPagination(query, opts.Page, opts.PerPage)
	opts.addFilterParams(query)
	opts.addArrayParams(query)

	return query
}

// addFilterParams adds filter query parameters.
func (opts *ResourcesListOptions) addFilterParams(query url.Values) {
	addStringSlice(query, "status", opts.Status)
	addStringSlice(query, "technology", opts.Technology)
	addStringSlice(query, "provider", opts.Provider)
	addStringSlice(query, "type", opts.Type)
	addStringSlice(query, "repository", opts.Repository)
	addStringSlice(query, "target", opts.Target)
	addStringSlice(query, "extracted_account", opts.ExtractedAccount)
	addBoolSlice(query, "is_archived", opts.IsArchived)
	addStringSlice(query, "policy_severity", opts.PolicySeverity)
	addString(query, "search", opts.Search)
	if opts.StackID > 0 {
		query.Set("stack_id", strconv.Itoa(opts.StackID))
	}
}

// addArrayParams adds array query parameters that use query.Add.
func (opts *ResourcesListOptions) addArrayParams(query url.Values) {
	for _, sort := range opts.Sort {
		query.Add("sort", sort)
	}
}

// List retrieves resources for an organization with optional filters.
//
// GET /v1/resources/{org_uuid}
//
// Resources are stack-level entities (e.g. Terraform resources) synced from plans/state.
// Use filters to narrow by stack, status, technology, provider, type, repository, target, etc.
func (s *ResourcesService) List(ctx context.Context, orgUUID string, opts *ResourcesListOptions) (*ResourcesListResponse, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}

	path := fmt.Sprintf("/v1/resources/%s", orgUUID)

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

	var result ResourcesListResponse
	resp, err := s.client.do(req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}

// Get retrieves a specific resource by UUID (includes details such as values when available).
//
// GET /v1/resources/{org_uuid}/{resource_uuid}
func (s *ResourcesService) Get(ctx context.Context, orgUUID, resourceUUID string) (*Resource, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}
	if resourceUUID == "" {
		return nil, nil, fmt.Errorf("resource UUID is required")
	}

	path := fmt.Sprintf("/v1/resources/%s/%s", orgUUID, resourceUUID)

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var resource Resource
	resp, err := s.client.do(req, &resource)
	if err != nil {
		return nil, resp, err
	}

	return &resource, resp, nil
}
