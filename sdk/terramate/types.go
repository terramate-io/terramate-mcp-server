package terramate

import "time"

// Organization represents a Terramate Cloud organization
type Organization struct {
	UUID        string    `json:"org_uuid"`
	Name        string    `json:"org_name"`
	DisplayName string    `json:"org_display_name"`
	Domain      string    `json:"org_domain,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// User represents a Terramate Cloud user
type User struct {
	UUID      string    `json:"user_uuid"`
	Email     string    `json:"email"`
	Name      string    `json:"display_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Membership represents a user's membership in an organization
// Maps to GetMembershipsResponseObject in the OpenAPI spec
type Membership struct {
	MemberID       int    `json:"member_id"`
	OrgUUID        string `json:"org_uuid"`
	OrgName        string `json:"org_name"`
	OrgDisplayName string `json:"org_display_name"`
	OrgDomain      string `json:"org_domain,omitempty"`
	Role           string `json:"role"`   // admin or member
	Status         string `json:"status"` // active, inactive, invited, sso_invited, trusted
}

// PaginatedResult represents pagination information from API responses
// Maps to PaginatedResultObject in the OpenAPI spec
type PaginatedResult struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// HasNextPage returns true if there are more pages after the current one
func (p *PaginatedResult) HasNextPage() bool {
	if p.PerPage == 0 {
		return false
	}
	totalPages := (p.Total + p.PerPage - 1) / p.PerPage
	return p.Page < totalPages
}

// HasPrevPage returns true if there are pages before the current one
func (p *PaginatedResult) HasPrevPage() bool {
	return p.Page > 1
}

// TotalPages returns the total number of pages
func (p *PaginatedResult) TotalPages() int {
	if p.PerPage == 0 {
		return 0
	}
	return (p.Total + p.PerPage - 1) / p.PerPage
}

// ListOptions represents common list options
type ListOptions struct {
	Page    int
	PerPage int
}
