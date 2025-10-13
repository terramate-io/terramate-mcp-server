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
	if p.PerPage == 0 || p.Page < 1 {
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

// Stack represents a Terramate Cloud stack
// Maps to Stack in the OpenAPI spec
type Stack struct {
	StackID              int             `json:"stack_id"`
	Repository           string          `json:"repository"`
	Target               string          `json:"target,omitempty"`
	Path                 string          `json:"path"`
	DefaultBranch        string          `json:"default_branch"`
	MetaID               string          `json:"meta_id"`
	MetaName             string          `json:"meta_name,omitempty"`
	MetaDescription      string          `json:"meta_description,omitempty"`
	MetaTags             []string        `json:"meta_tags,omitempty"`
	Status               string          `json:"status"` // canceled, drifted, failed, ok, unknown
	DeploymentStatus     string          `json:"deployment_status"`
	DriftStatus          string          `json:"drift_status"` // ok, drifted, failed, unknown
	Draft                bool            `json:"draft"`
	IsArchived           bool            `json:"is_archived"`
	ArchivedAt           *time.Time      `json:"archived_at,omitempty"`
	ArchivedByUserUUID   string          `json:"archived_by_user_uuid,omitempty"`
	UnarchivedAt         *time.Time      `json:"unarchived_at,omitempty"`
	UnarchivedByUserUUID string          `json:"unarchived_by_user_uuid,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	SeenAt               time.Time       `json:"seen_at"`
	RelatedStacks        []RelatedStack  `json:"related_stacks,omitempty"`
	Resources            *StackResources `json:"resources,omitempty"`
}

// RelatedStack represents a stack from other targets with the same repository and meta_id
// Only set when getting a single stack
type RelatedStack struct {
	StackID int    `json:"stack_id"`
	Target  string `json:"target"`
}

// StackResources represents resources related data for a stack
// Only set when getting the stack list and stack details
type StackResources struct {
	Count       int               `json:"count"`
	PolicyCheck *StackPolicyCheck `json:"policy_check,omitempty"`
}

// StackPolicyCheck represents policy check results from a stack
type StackPolicyCheck struct {
	CreatedAt time.Time           `json:"created_at"`
	Passed    bool                `json:"passed"`
	Counters  PolicyCheckCounters `json:"counters"`
}

// PolicyCheckCounters represents counters for policy check results
type PolicyCheckCounters struct {
	PassedCount         int `json:"passed_count"`
	SeverityLowCount    int `json:"severity_low_count"`
	SeverityMediumCount int `json:"severity_medium_count"`
	SeverityHighCount   int `json:"severity_high_count"`
}

// StacksListResponse represents the response from listing stacks
// Maps to GetStacksResponseObject in the OpenAPI spec
type StacksListResponse struct {
	Stacks          []Stack         `json:"stacks"`
	PaginatedResult PaginatedResult `json:"paginated_result"`
}

// StacksListOptions represents options for listing stacks
type StacksListOptions struct {
	ListOptions
	// Repository filters by exact repository URLs (e.g., "github.com/owner/repo")
	// Only full string matches are supported (no substring or pattern matching)
	Repository       []string
	Target           []string
	Status           []string
	DeploymentStatus []string
	DriftStatus      []string
	Draft            *bool
	IsArchived       []bool
	// Search performs substring search on meta_id, meta_name, meta_description, and path
	Search string
	MetaID string
	// DeploymentUUID filters stacks by deployment UUID
	DeploymentUUID string
	MetaTag        []string
	// PolicySeverity filters by policy check results
	// Valid values: missing, none, passed, low, medium, high
	PolicySeverity []string
	Sort           []string
}
