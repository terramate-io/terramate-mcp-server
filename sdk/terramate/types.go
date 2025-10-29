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
	SeenAt               *time.Time      `json:"seen_at,omitempty"`
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

// UserInfo represents user information in drift/deployment contexts
// Maps to UserInfo in the OpenAPI spec
type UserInfo struct {
	DisplayName    string `json:"display_name,omitempty"`
	Position       string `json:"position,omitempty"`
	UserPictureURL string `json:"user_picture_url,omitempty"`
}

// TrustInfo represents trust information for CI/CD authentication
// Maps to TrustInfo in the OpenAPI spec
type TrustInfo struct {
	AuthID string `json:"auth_id"`
}

// ChangesetDetails represents the details of a terraform/tofu plan changeset
// Maps to ChangesetDetails in the OpenAPI spec
type ChangesetDetails struct {
	Provisioner    string `json:"provisioner,omitempty"`     // terraform, opentofu
	Serial         int64  `json:"serial,omitempty"`          // Terraform state serial number
	ChangesetASCII string `json:"changeset_ascii,omitempty"` // ASCII format plan (up to 4MB)
	ChangesetJSON  string `json:"changeset_json,omitempty"`  // JSON format plan (up to 16MB)
}

// Drift represents a drift detection run for a stack
// Maps to Drift in the OpenAPI spec
// Note: drift_details and stack are not populated in responses that list drifts
type Drift struct {
	ID           int                    `json:"id"`
	OrgUUID      string                 `json:"org_uuid"`
	StackID      int                    `json:"stack_id"`
	Status       string                 `json:"status"` // ok, drifted, failed
	Metadata     map[string]interface{} `json:"metadata"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	FinishedAt   *time.Time             `json:"finished_at,omitempty"`
	AuthType     string                 `json:"auth_type,omitempty"` // gha, gitlabcicd, idp, tmco
	AuthUser     *UserInfo              `json:"auth_user,omitempty"`
	AuthTrust    *TrustInfo             `json:"auth_trust,omitempty"`
	Stack        *Stack                 `json:"stack,omitempty"`
	DriftDetails *ChangesetDetails      `json:"drift_details,omitempty"` // Only populated when getting specific drift
	GroupingKey  string                 `json:"grouping_key,omitempty"`
	Cmd          []string               `json:"cmd,omitempty"`
}

// DriftsListResponse represents the response from listing drifts
// Maps to GetDriftsResponseObject in the OpenAPI spec
type DriftsListResponse struct {
	Drifts          []Drift         `json:"drifts,omitempty"`
	PaginatedResult PaginatedResult `json:"paginated_result"`
}

// DriftsListOptions represents options for listing drifts
type DriftsListOptions struct {
	ListOptions
	// DriftStatus filters by drift status (ok, drifted, failed)
	DriftStatus []string
	// GroupingKey filters by grouping key
	GroupingKey string
}

// VCSLabel represents a label on a pull/merge request
// Maps to VCSLabel in the OpenAPI spec
type VCSLabel struct {
	Name        string `json:"name"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
}

// ResourceChangesActionsSummary represents a summary of resource changes in a preview
// Maps to ResourceChangesActionsSummary in the OpenAPI spec
type ResourceChangesActionsSummary struct {
	CreateCount  int `json:"create_count"`
	DeleteCount  int `json:"delete_count"`
	NoopCount    int `json:"noop_count"`
	ReadCount    int `json:"read_count"`
	ReplaceCount int `json:"replace_count"`
	UpdateCount  int `json:"update_count"`
	ImportCount  int `json:"import_count"`
	MoveCount    int `json:"move_count"`
	ForgetCount  int `json:"forget_count"`
}

// Preview represents a preview summary for a review request
// Maps to Preview in the OpenAPI spec
type Preview struct {
	ID              int                            `json:"id"`
	Status          string                         `json:"status"` // current, outdated
	AffectedCount   int                            `json:"affected_count"`
	PendingCount    int                            `json:"pending_count"`
	RunningCount    int                            `json:"running_count"`
	ChangedCount    int                            `json:"changed_count"`
	UnchangedCount  int                            `json:"unchanged_count"`
	FailedCount     int                            `json:"failed_count"`
	CanceledCount   int                            `json:"canceled_count"`
	ResourceChanges *ResourceChangesActionsSummary `json:"resource_changes,omitempty"`
}

// ReviewRequestCollaborator represents a collaborator on a review request
type ReviewRequestCollaborator struct {
	ID          int      `json:"id"`
	DisplayName string   `json:"display_name"`
	AvatarURL   string   `json:"avatar_url,omitempty"`
	Platform    string   `json:"platform"`
	Roles       []string `json:"roles"` // author, reviewer, requested_reviewer
}

// ReviewRequest represents a pull/merge request in Terramate Cloud
// Maps to ReviewRequest in the OpenAPI spec
type ReviewRequest struct {
	ReviewRequestID       int                         `json:"review_request_id"`
	Platform              string                      `json:"platform,omitempty"` // github, gitlab, bitbucket
	Repository            string                      `json:"repository,omitempty"`
	CommitSHA             string                      `json:"commit_sha,omitempty"`
	Number                int                         `json:"number,omitempty"`
	Title                 string                      `json:"title,omitempty"`
	Description           string                      `json:"description,omitempty"`
	URL                   string                      `json:"url,omitempty"`
	Status                string                      `json:"status,omitempty"` // open, merged, closed, approved, changes_requested, review_required
	PlatformUpdatedAt     *time.Time                  `json:"platform_updated_at,omitempty"`
	PlatformCreatedAt     *time.Time                  `json:"platform_created_at,omitempty"`
	PlatformPushedAt      *time.Time                  `json:"platform_pushed_at,omitempty"`
	PlatformMergedAt      *time.Time                  `json:"platform_merged_at,omitempty"`
	Branch                string                      `json:"branch,omitempty"`
	BaseBranch            string                      `json:"base_branch,omitempty"`
	Draft                 bool                        `json:"draft,omitempty"`
	ReviewDecision        string                      `json:"review_decision,omitempty"` // approved, changes_requested, review_required, none
	ChangesRequestedCount int                         `json:"changes_requested_count,omitempty"`
	ApprovedCount         int                         `json:"approved_count,omitempty"`
	ChecksTotalCount      int                         `json:"checks_total_count,omitempty"`
	ChecksFailureCount    int                         `json:"checks_failure_count,omitempty"`
	ChecksSuccessCount    int                         `json:"checks_success_count,omitempty"`
	Labels                []VCSLabel                  `json:"labels,omitempty"`
	Preview               *Preview                    `json:"preview,omitempty"`
	Collaborators         []ReviewRequestCollaborator `json:"collaborators,omitempty"`
}

// ResourceChanges represents resource changes in a stack preview
// Maps to ResourceChanges in the OpenAPI spec
type ResourceChanges struct {
	ActionsSummary ResourceChangesActionsSummary `json:"actions_summary"`
}

// StackPreview represents a terraform plan preview for a single stack
// Maps to StackPreview in the OpenAPI spec
//
// This type is used when getting a review request and includes the FULL
// terraform plan content in changeset_details (up to 4MB ASCII plan).
//
// Use this when: Getting PR details with ReviewRequests.Get()
// The changeset_details field contains the actual terraform plan output.
type StackPreview struct {
	StackPreviewID   int               `json:"stack_preview_id"`
	Status           string            `json:"status"` // affected, pending, running, changed, unchanged, failed, canceled
	Path             string            `json:"path,omitempty"`
	Technology       string            `json:"technology"` // terraform, opentofu, other
	TechnologyLayer  string            `json:"technology_layer,omitempty"`
	UpdatedAt        time.Time         `json:"updated_at"`
	Stack            *Stack            `json:"stack,omitempty"`
	ChangesetDetails *ChangesetDetails `json:"changeset_details,omitempty"`
	ResourceChanges  *ResourceChanges  `json:"resource_changes,omitempty"`
}

// ReviewRequestGetResponse represents the response from getting a review request
// Maps to GetReviewRequestResponse in the OpenAPI spec
type ReviewRequestGetResponse struct {
	ReviewRequest ReviewRequest  `json:"review_request"`
	StackPreviews []StackPreview `json:"stack_previews,omitempty"`
}

// ReviewRequestsListResponse represents the response from listing review requests
// Maps to GetReviewRequestsResponse in the OpenAPI spec
type ReviewRequestsListResponse struct {
	ReviewRequests  []ReviewRequest `json:"review_requests"`
	PaginatedResult PaginatedResult `json:"paginated_result"`
}

// ReviewRequestsListOptions represents options for listing review requests
type ReviewRequestsListOptions struct {
	ListOptions
	Status          []string // open, merged, closed, approved, changes_requested, review_required
	Repository      []string
	CollaboratorID  []int
	UserUUID        []string
	AuthorUUID      []string
	ReviewRequested []string // User UUIDs of requested reviewers
	Draft           *bool
	Search          string // Searches PR number, title, commit SHA, branch
	CreatedAtFrom   *time.Time
	CreatedAtTo     *time.Time
	Sort            []string
}

// ReviewRequestGetOptions represents options for getting a review request
type ReviewRequestGetOptions struct {
	ExcludeStackPreviews bool
}

// WorkflowDeploymentGroup represents a CI/CD workflow deployment run
// Maps to WorkflowDeploymentGroup in the OpenAPI spec
type WorkflowDeploymentGroup struct {
	ID                        int                    `json:"id"`
	Status                    string                 `json:"status"` // ok, failed, processing
	CommitTitle               string                 `json:"commit_title"`
	CommitSHA                 string                 `json:"commit_sha,omitempty"`
	Repository                string                 `json:"repository"`
	AuthType                  string                 `json:"auth_type,omitempty"` // gha, gitlabcicd, idp, tmco
	AuthID                    string                 `json:"auth_id,omitempty"`
	AuthUser                  *UserInfo              `json:"auth_user,omitempty"`
	CanceledCount             int                    `json:"canceled_count"`
	FailedCount               int                    `json:"failed_count"`
	OkCount                   int                    `json:"ok_count"`
	PendingCount              int                    `json:"pending_count"`
	RunningCount              int                    `json:"running_count"`
	StackDeploymentTotalCount int                    `json:"stack_deployment_total_count"`
	CreatedAt                 time.Time              `json:"created_at"`
	StartedAt                 *time.Time             `json:"started_at,omitempty"`
	FinishedAt                *time.Time             `json:"finished_at,omitempty"`
	Metadata                  map[string]interface{} `json:"metadata,omitempty"`
	ReviewRequest             *ReviewRequest         `json:"review_request,omitempty"`
	Branch                    string                 `json:"branch,omitempty"`
	WorkflowName              string                 `json:"workflow_name,omitempty"`
	GroupingKey               string                 `json:"grouping_key,omitempty"`
}

// StackDeployment represents a deployment of a single stack
// Maps to StackDeployment in the OpenAPI spec
type StackDeployment struct {
	ID               int               `json:"id"`
	DeploymentUUID   string            `json:"deployment_uuid"`
	Path             string            `json:"path"`
	Cmd              []string          `json:"cmd"`
	Status           string            `json:"status"` // canceled, failed, ok, pending, running
	CreatedAt        time.Time         `json:"created_at"`
	StartedAt        *time.Time        `json:"started_at,omitempty"`
	FinishedAt       *time.Time        `json:"finished_at,omitempty"`
	FixedAt          *time.Time        `json:"fixed_at,omitempty"`
	Stack            *Stack            `json:"stack,omitempty"`
	ChangesetDetails *ChangesetDetails `json:"changeset_details,omitempty"`
}

// DeploymentsListResponse represents the response from listing workflow deployments
// Maps to GetOrganizationDeploymentsResponseObject in the OpenAPI spec
type DeploymentsListResponse struct {
	Deployments     []WorkflowDeploymentGroup `json:"deployments"`
	PaginatedResult PaginatedResult           `json:"paginated_result"`
}

// StackDeploymentsListResponse represents the response from listing stack deployments
// Maps to StackDeploymentsCollection in the OpenAPI spec
type StackDeploymentsListResponse struct {
	StackDeployments []StackDeployment `json:"stack_deployments"`
	PaginatedResult  PaginatedResult   `json:"paginated_result"`
}

// DeploymentsListOptions represents options for listing workflow deployments
type DeploymentsListOptions struct {
	ListOptions
	Repository     []string
	AuthType       []string // gha, gitlabcicd, idp, tmco
	Status         []string // ok, failed, processing
	CollaboratorID []int
	UserUUID       []string
	Search         string
	CreatedAtFrom  *time.Time
	CreatedAtTo    *time.Time
	StartedAtFrom  *time.Time
	StartedAtTo    *time.Time
	FinishedAtFrom *time.Time
	FinishedAtTo   *time.Time
	Sort           []string
}

// StackDeploymentsListOptions represents options for listing stack deployments
type StackDeploymentsListOptions struct {
	ListOptions
	Status        []string // canceled, failed, ok, pending, running
	CreatedAtFrom *time.Time
	CreatedAtTo   *time.Time
}

// CommandLogLine represents a single log line from terraform/tofu output
// Maps to CommandLogLine in the OpenAPI spec
type CommandLogLine struct {
	LogLine   int       `json:"log_line"`
	Timestamp time.Time `json:"timestamp"`
	Channel   string    `json:"channel"` // stdout, stderr
	Message   string    `json:"message"`
}

// StackPreviewLogsResponse represents the response from getting stack preview logs
// Maps to GetStackPreviewLogsResponse in the OpenAPI spec
type StackPreviewLogsResponse struct {
	StackPreviewLogLines []CommandLogLine `json:"stack_preview_log_lines"`
	PaginatedResult      PaginatedResult  `json:"paginated_result"`
}

// Summary represents an AI-generated summary
// Maps to SummaryResponse.summary in the OpenAPI spec
type Summary struct {
	Contents  []string  `json:"contents"`
	CreatedAt time.Time `json:"created_at"`
	Requester *User     `json:"requester,omitempty"`
}

// SummaryResponse represents an AI summary response
// Maps to SummaryResponse in the OpenAPI spec
type SummaryResponse struct {
	Summary Summary `json:"summary"`
}

// StackPreviewV2 represents preview metadata without full plan content
// Maps to StackPreviewV2 in the OpenAPI spec
//
// This type provides metadata (sizes, counts, timestamps) but does NOT
// include the full terraform plan in changeset_details. Instead, it provides:
// - changeset_ascii_size: Size of the plan (not the content)
// - logs_stderr_count: Number of error log lines (not the logs)
// - logs_stdout_count: Number of output log lines (not the logs)
//
// To get the actual logs, use Previews.GetLogs().
// To get the full plan, use the changeset endpoint (not yet implemented).
//
// Use this when: Getting preview details with Previews.Get()
// This is useful for checking preview status without loading large plan content.
type StackPreviewV2 struct {
	ID                   int                            `json:"id"`
	CreatedAt            time.Time                      `json:"created_at"`
	UpdatedAt            time.Time                      `json:"updated_at"`
	CommitSHA            string                         `json:"commit_sha"`
	ReviewRequestID      int                            `json:"review_request_id"`
	Status               string                         `json:"status"` // affected, pending, running, changed, unchanged, failed, canceled
	StackID              int                            `json:"stack_id"`
	Technology           string                         `json:"technology,omitempty"`
	TechnologyLayer      string                         `json:"technology_layer,omitempty"`
	Path                 string                         `json:"path,omitempty"`
	Stack                *Stack                         `json:"stack,omitempty"`
	ChangesetProvisioner string                         `json:"changeset_provisioner,omitempty"`
	ChangesetJSONSize    int64                          `json:"changeset_json_size,omitempty"`
	ChangesetASCIISize   int64                          `json:"changeset_ascii_size,omitempty"`
	LogsStderrCount      int64                          `json:"logs_stderr_count,omitempty"`
	LogsStdoutCount      int64                          `json:"logs_stdout_count,omitempty"`
	ChangesetActions     *ResourceChangesActionsSummary `json:"changeset_actions,omitempty"`
}

// PreviewLogsOptions represents options for getting preview logs
type PreviewLogsOptions struct {
	ListOptions
	Channel string // stdout, stderr
}

// DeploymentLogsResponse represents the response from getting deployment logs
// Maps to GetDeploymentLogsResponseObject in the OpenAPI spec
type DeploymentLogsResponse struct {
	DeploymentLogLines []CommandLogLine `json:"deployment_log_lines"`
	PaginatedResult    PaginatedResult  `json:"paginated_result"`
}

// DeploymentLogsOptions represents options for getting deployment logs
type DeploymentLogsOptions struct {
	ListOptions
	Channel string // stdout, stderr
}
