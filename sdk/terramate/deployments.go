package terramate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DeploymentsService handles communication with the deployments related
// methods of the Terramate Cloud API
type DeploymentsService struct {
	client *Client
}

// buildQuery constructs URL query parameters from DeploymentsListOptions
func (opts *DeploymentsListOptions) buildQuery() url.Values {
	query := url.Values{}
	if opts == nil {
		return query
	}

	addPagination(query, opts.Page, opts.PerPage)
	addStringSlice(query, "repository", opts.Repository)
	addStringSlice(query, "auth_type", opts.AuthType)
	addStringSlice(query, "status", opts.Status)
	addIntSlice(query, "collaborator_id", opts.CollaboratorID)
	addStringSlice(query, "user_uuid", opts.UserUUID)
	addString(query, "search", opts.Search)
	addTimePtr(query, "created_at_from", opts.CreatedAtFrom)
	addTimePtr(query, "created_at_to", opts.CreatedAtTo)
	addTimePtr(query, "started_at_from", opts.StartedAtFrom)
	addTimePtr(query, "started_at_to", opts.StartedAtTo)
	addTimePtr(query, "finished_at_from", opts.FinishedAtFrom)
	addTimePtr(query, "finished_at_to", opts.FinishedAtTo)

	for _, sort := range opts.Sort {
		query.Add("sort", sort)
	}

	return query
}

// buildQuery constructs URL query parameters from StackDeploymentsListOptions
func (opts *StackDeploymentsListOptions) buildQuery() url.Values {
	query := url.Values{}
	if opts == nil {
		return query
	}

	addPagination(query, opts.Page, opts.PerPage)
	addStringSlice(query, "status", opts.Status)
	addTimePtr(query, "created_at_from", opts.CreatedAtFrom)
	addTimePtr(query, "created_at_to", opts.CreatedAtTo)

	return query
}

// List retrieves all workflow deployment groups for an organization.
//
// GET /v1/organizations/{org_uuid}/deployments
//
// This endpoint returns workflow deployments (CI/CD runs) matching the provided filters.
// Results are sorted by updated_at in descending order by default.
//
// Access: Members of the organization with any role are allowed to query.
func (s *DeploymentsService) List(ctx context.Context, orgUUID string, opts *DeploymentsListOptions) (*DeploymentsListResponse, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}

	path := fmt.Sprintf("/v1/organizations/%s/deployments", orgUUID)

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

	var result DeploymentsListResponse
	resp, err := s.client.do(req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}

// GetWorkflow retrieves a specific workflow deployment group by ID.
//
// GET /v1/workflow_deployment_groups/{org_uuid}/{workflow_deployment_group_id}
//
// This endpoint returns details for a specific workflow deployment group (CI/CD run).
//
// Access: All members of the organization with any role are allowed to query.
func (s *DeploymentsService) GetWorkflow(ctx context.Context, orgUUID string, workflowDeploymentGroupID int) (*WorkflowDeploymentGroup, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}
	if workflowDeploymentGroupID <= 0 {
		return nil, nil, fmt.Errorf("workflow deployment group ID must be positive")
	}

	path := fmt.Sprintf("/v1/workflow_deployment_groups/%s/%d", orgUUID, workflowDeploymentGroupID)

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var workflow WorkflowDeploymentGroup
	resp, err := s.client.do(req, &workflow)
	if err != nil {
		return nil, resp, err
	}

	return &workflow, resp, nil
}

// ListForWorkflow retrieves all stack deployments for a workflow deployment group.
//
// GET /v1/workflow_deployment_groups/{org_uuid}/{workflow_deployment_group_id}/stacks
//
// This endpoint returns all stack deployments that are part of a workflow deployment group.
//
// Access: All members of the organization with any role are allowed to query.
func (s *DeploymentsService) ListForWorkflow(ctx context.Context, orgUUID string, workflowDeploymentGroupID int, opts *ListOptions) (*StackDeploymentsListResponse, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}
	if workflowDeploymentGroupID <= 0 {
		return nil, nil, fmt.Errorf("workflow deployment group ID must be positive")
	}

	path := fmt.Sprintf("/v1/workflow_deployment_groups/%s/%d/stacks", orgUUID, workflowDeploymentGroupID)

	if opts != nil {
		query := url.Values{}
		addPagination(query, opts.Page, opts.PerPage)
		if len(query) > 0 {
			path = path + "?" + query.Encode()
		}
	}

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result StackDeploymentsListResponse
	resp, err := s.client.do(req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}

// ListStackDeployments retrieves all stack deployments for an organization.
//
// GET /v1/stack_deployments/{org_uuid}
//
// This endpoint returns stack deployments across the entire organization.
//
// Access: All members of the organization with any role are allowed to query.
func (s *DeploymentsService) ListStackDeployments(ctx context.Context, orgUUID string, opts *StackDeploymentsListOptions) (*StackDeploymentsListResponse, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}

	path := fmt.Sprintf("/v1/stack_deployments/%s", orgUUID)

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

	var result StackDeploymentsListResponse
	resp, err := s.client.do(req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}

// GetStackDeployment retrieves a specific stack deployment by ID.
//
// GET /v1/stack_deployments/{org_uuid}/{stack_deployment_id}
//
// This endpoint returns details for a specific stack deployment including
// the terraform plan (changeset_details).
//
// Access: All members of the organization with any role are allowed to query.
func (s *DeploymentsService) GetStackDeployment(ctx context.Context, orgUUID string, stackDeploymentID int) (*StackDeployment, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}
	if stackDeploymentID <= 0 {
		return nil, nil, fmt.Errorf("stack deployment ID must be positive")
	}

	path := fmt.Sprintf("/v1/stack_deployments/%s/%d", orgUUID, stackDeploymentID)

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var deployment StackDeployment
	resp, err := s.client.do(req, &deployment)
	if err != nil {
		return nil, resp, err
	}

	return &deployment, resp, nil
}

// GetDeploymentLogs retrieves terraform command logs for a stack deployment.
//
// GET /v1/stacks/{org_uuid}/{stack_id}/deployments/{deployment_uuid}/logs
//
// This endpoint returns the terraform apply/destroy command output logs
// which are essential for debugging failed deployments.
//
// Access: All members of the organization with any role are allowed to query.
func (s *DeploymentsService) GetDeploymentLogs(ctx context.Context, orgUUID string, stackID int, deploymentUUID string, opts *DeploymentLogsOptions) (*DeploymentLogsResponse, *Response, error) {
	if orgUUID == "" {
		return nil, nil, fmt.Errorf("organization UUID is required")
	}
	if stackID <= 0 {
		return nil, nil, fmt.Errorf("stack ID must be positive")
	}
	if deploymentUUID == "" {
		return nil, nil, fmt.Errorf("deployment UUID is required")
	}

	path := fmt.Sprintf("/v1/stacks/%s/%d/deployments/%s/logs", orgUUID, stackID, deploymentUUID)

	if opts != nil {
		query := url.Values{}
		addPagination(query, opts.Page, opts.PerPage)
		addString(query, "channel", opts.Channel)
		if len(query) > 0 {
			path = path + "?" + query.Encode()
		}
	}

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result DeploymentLogsResponse
	resp, err := s.client.do(req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}
