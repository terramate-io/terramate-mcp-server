package terramate

import (
	"context"
	"fmt"
)

// MembershipsService handles communication with the memberships related
// methods of the Terramate Cloud API
type MembershipsService struct {
	client *Client
}

// Retrieves the organization membership for the authenticated user
//
// GET /v1/memberships
//
// This endpoint returns the organizations the current user belongs to.
//
// Note: API keys are bound to specific organizations, so when using API key
// authentication, this will typically return only one membership.
func (s *MembershipsService) List(ctx context.Context) ([]Membership, *Response, error) {
	path := "/v1/memberships"

	req, err := s.client.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// API returns an array directly, not wrapped in an object
	var memberships []Membership
	resp, err := s.client.do(req, &memberships)
	if err != nil {
		return nil, resp, err
	}

	return memberships, resp, nil
}
