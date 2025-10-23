package terramate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ReviewRequestsService handles communication with the review requests related
// methods of the Terramate Cloud API
type ReviewRequestsService struct {
	client *Client
}

// buildQuery constructs URL query parameters from ReviewRequestsListOptions
func (opts *ReviewRequestsListOptions) buildQuery() url.Values {
	query := url.Values{}
	if opts == nil {
		return query
	}

	addPagination(query, opts.Page, opts.PerPage)
	addStringSlice(query, "status", opts.Status)
	addStringSlice(query, "repository", opts.Repository)
	addString(query, "search", opts.Search)
	addBoolPtr(query, "draft", opts.Draft)
	addIntSlice(query, "collaborator_id", opts.CollaboratorID)
	addStringSlice(query, "user_uuid", opts.UserUUID)
	addStringSlice(query, "author_uuid", opts.AuthorUUID)
	addStringSlice(query, "review_requested_uuid", opts.ReviewRequested)
	addTimePtr(query, "created_at_from", opts.CreatedAtFrom)
	addTimePtr(query, "created_at_to", opts.CreatedAtTo)

	// Add sort parameters (use query.Add for multiple values)
	for _, sort := range opts.Sort {
		query.Add("sort", sort)
	}

	return query
}

// List retrieves all review requests for an organization.
//
// GET /v1/review_requests/{org_uuid}
//
// This endpoint returns review requests (pull/merge requests) matching the provided filters.
// Results are sorted by updated_at in descending order by default.
//
// Access: Members of the organization with any role are allowed to query.
func (s *ReviewRequestsService) List(ctx context.Context, orgUUID string, opts *ReviewRequestsListOptions) (*ReviewRequestsListResponse, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}

	path := fmt.Sprintf("/v1/review_requests/%s", orgUUID)

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

	var result ReviewRequestsListResponse
	resp, err := s.client.do(req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}

// Get retrieves a specific review request by ID with optional stack previews.
//
// GET /v1/review_requests/{org_uuid}/{review_request_id}
//
// This endpoint returns details for a specific review request including
// the latest available preview for affected stacks (unless excluded).
//
// Access: All members of the organization with any role are allowed to query.
func (s *ReviewRequestsService) Get(ctx context.Context, orgUUID string, reviewRequestID int, opts *ReviewRequestGetOptions) (*ReviewRequestGetResponse, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}
	if reviewRequestID <= 0 {
		return nil, nil, fmt.Errorf("review request ID must be positive")
	}

	path := fmt.Sprintf("/v1/review_requests/%s/%d", orgUUID, reviewRequestID)

	// Add query parameters
	if opts != nil && opts.ExcludeStackPreviews {
		path += "?exclude_stack_previews=true"
	}

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result ReviewRequestGetResponse
	resp, err := s.client.do(req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}
