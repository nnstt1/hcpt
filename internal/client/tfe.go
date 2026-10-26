package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"

	tfe "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	"github.com/hashicorp/go-tfe/v2/api/runs"
	runsitem "github.com/hashicorp/go-tfe/v2/api/runs/item"
	"github.com/hashicorp/go-tfe/v2/api/workspaces"
	abs "github.com/microsoft/kiota-abstractions-go"
)

func (c *ClientWrapper) ListOrganizations(ctx context.Context, opts *OrganizationListOptions) (*OrganizationList, error) {
	params := &organizations.OrganizationsRequestBuilderGetQueryParameters{}
	if opts != nil {
		params.Pagenumber = int32Ptr(opts.PageNumber)
		params.Pagesize = int32Ptr(opts.PageSize)
	}

	resp, err := c.client.API.Organizations().Get(ctx, &abs.RequestConfiguration[organizations.OrganizationsRequestBuilderGetQueryParameters]{
		QueryParameters: params,
	})
	if err != nil {
		return nil, err
	}

	list := &OrganizationList{}
	if resp == nil {
		return list, nil
	}
	if meta := resp.GetMeta(); meta != nil {
		list.Pagination = toPagination(meta.GetPagination())
	}
	for _, o := range resp.GetData() {
		list.Items = append(list.Items, toOrganization(o))
	}
	return list, nil
}

func (c *ClientWrapper) ReadOrganization(ctx context.Context, org string) (*Organization, error) {
	resp, err := c.client.API.Organizations().ByOrganization_name(org).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.GetData() == nil {
		return nil, fmt.Errorf("organization %q: empty response", org)
	}
	return toOrganization(resp.GetData()), nil
}

// ReadEntitlements reads the entitlement set of an organization.
// go-tfe v2 does not provide the entitlement-set endpoint (and its model has
// no typed attributes), so the endpoint is called via raw HTTP.
func (c *ClientWrapper) ReadEntitlements(ctx context.Context, org string) (*Entitlements, error) {
	address := c.address
	if address == "" {
		address = "https://app.terraform.io"
	}

	apiURL := strings.TrimRight(address, "/") + "/api/v2/organizations/" + url.PathEscape(org) + "/entitlement-set"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch entitlements: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("entitlement-set endpoint returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read entitlements response: %w", err)
	}

	return parseEntitlementsResponse(body)
}

// parseEntitlementsResponse extracts entitlements from the JSON:API response.
func parseEntitlementsResponse(body []byte) (*Entitlements, error) {
	var response struct {
		Data struct {
			Attributes struct {
				Agents                bool `json:"agents"`
				AuditLogging          bool `json:"audit-logging"`
				CostEstimation        bool `json:"cost-estimation"`
				Operations            bool `json:"operations"`
				PrivateModuleRegistry bool `json:"private-module-registry"`
				RunTasks              bool `json:"run-tasks"`
				SSO                   bool `json:"sso"`
				Sentinel              bool `json:"sentinel"`
				StateStorage          bool `json:"state-storage"`
				Teams                 bool `json:"teams"`
				VCSIntegrations       bool `json:"vcs-integrations"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse entitlements response: %w", err)
	}

	a := response.Data.Attributes
	return &Entitlements{
		Agents:                a.Agents,
		AuditLogging:          a.AuditLogging,
		CostEstimation:        a.CostEstimation,
		Operations:            a.Operations,
		PrivateModuleRegistry: a.PrivateModuleRegistry,
		RunTasks:              a.RunTasks,
		SSO:                   a.SSO,
		Sentinel:              a.Sentinel,
		StateStorage:          a.StateStorage,
		Teams:                 a.Teams,
		VCSIntegrations:       a.VCSIntegrations,
	}, nil
}

func (c *ClientWrapper) ListWorkspaces(ctx context.Context, org string, opts *WorkspaceListOptions) (*WorkspaceList, error) {
	params := &organizations.ItemWorkspacesRequestBuilderGetQueryParameters{}
	if opts != nil {
		params.Pagenumber = int32Ptr(opts.PageNumber)
		params.Pagesize = int32Ptr(opts.PageSize)
		params.Filterprojectid = stringPtr(opts.ProjectID)
	}

	resp, err := c.client.API.Organizations().ByOrganization_name(org).Workspaces().Get(ctx, &abs.RequestConfiguration[organizations.ItemWorkspacesRequestBuilderGetQueryParameters]{
		QueryParameters: params,
	})
	if err != nil {
		return nil, err
	}

	list := &WorkspaceList{}
	if resp == nil {
		return list, nil
	}
	if meta := resp.GetMeta(); meta != nil {
		list.Pagination = toPagination(meta.GetPagination())
	}
	for _, ws := range resp.GetData() {
		list.Items = append(list.Items, toWorkspace(ws))
	}
	return list, nil
}

func (c *ClientWrapper) ReadWorkspace(ctx context.Context, org string, name string) (*Workspace, error) {
	resp, err := c.client.API.Organizations().ByOrganization_name(org).Workspaces().ByWorkspace_name(name).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.GetData() == nil {
		return nil, fmt.Errorf("workspace %q: empty response", name)
	}
	return toWorkspace(resp.GetData()), nil
}

func (c *ClientWrapper) ListRuns(ctx context.Context, workspaceID string, opts *RunListOptions) (*RunList, error) {
	params := &workspaces.ItemRunsRequestBuilderGetQueryParameters{}
	if opts != nil {
		params.Pagenumber = int32Ptr(opts.PageNumber)
		params.Pagesize = int32Ptr(opts.PageSize)
		params.Filteroperation = stringPtr(opts.Operation)
		params.Filterstatus = stringPtr(opts.Status)
	}

	resp, err := c.client.API.Workspaces().ByWorkspace_id(workspaceID).Runs().Get(ctx, &abs.RequestConfiguration[workspaces.ItemRunsRequestBuilderGetQueryParameters]{
		QueryParameters: params,
	})
	if err != nil {
		return nil, err
	}

	list := &RunList{}
	if resp == nil {
		return list, nil
	}
	if meta := resp.GetMeta(); meta != nil {
		list.Pagination = toPagination(meta.GetPagination())
	}
	for _, r := range resp.GetData() {
		list.Items = append(list.Items, toRun(r, nil))
	}
	return list, nil
}

func (c *ClientWrapper) ReadRun(ctx context.Context, runID string) (*Run, error) {
	return c.readRun(ctx, runID, runsitem.PLAN_GETINCLUDEQUERYPARAMETERTYPE)
}

func (c *ClientWrapper) ReadRunWithApply(ctx context.Context, runID string) (*Run, error) {
	return c.readRun(ctx, runID, runsitem.PLAN_GETINCLUDEQUERYPARAMETERTYPE, runsitem.APPLY_GETINCLUDEQUERYPARAMETERTYPE)
}

func (c *ClientWrapper) readRun(ctx context.Context, runID string, include ...runsitem.GetIncludeQueryParameterType) (*Run, error) {
	resp, err := c.client.API.Runs().ById(runID).Get(ctx, &abs.RequestConfiguration[runs.ItemRequestBuilderGetQueryParameters]{
		QueryParameters: &runs.ItemRequestBuilderGetQueryParameters{
			Include: include,
		},
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.GetData() == nil {
		return nil, fmt.Errorf("run %q: empty response", runID)
	}
	return toRun(resp.GetData(), resp.GetIncluded()), nil
}

// ReadPlanJSONOutput reads the JSON output of a plan.
// The endpoint responds with a redirect to a temporary download URL, which the
// go-tfe v2 typed API does not expose, so the raw response is streamed instead.
func (c *ClientWrapper) ReadPlanJSONOutput(ctx context.Context, planID string) ([]byte, error) {
	resp, err := c.client.GetStream(ctx, "/plans/"+url.PathEscape(planID)+"/json-output", nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plan json-output endpoint returned HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// ReadApplyLogs reads the log output for an apply.
func (c *ClientWrapper) ReadApplyLogs(ctx context.Context, applyID string) (io.Reader, error) {
	resp, err := c.client.API.Applies().ById(applyID).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	var logURL string
	if resp != nil && resp.GetData() != nil && resp.GetData().GetAttributes() != nil {
		logURL = deref(resp.GetData().GetAttributes().GetLogReadUrl())
	}
	if logURL == "" {
		return nil, fmt.Errorf("apply %q does not have a log URL", applyID)
	}

	// The log URL is a pre-signed archivist URL, so it is fetched without the API token.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, logURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	logResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch apply logs: %w", err)
	}
	defer func() { _ = logResp.Body.Close() }()

	if logResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apply log endpoint returned HTTP %d", logResp.StatusCode)
	}

	body, err := io.ReadAll(logResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read apply logs: %w", err)
	}
	return bytes.NewReader(body), nil
}

// ListVariables lists the variables of a workspace.
// The workspace vars endpoint is not paginated, so opts is ignored.
func (c *ClientWrapper) ListVariables(ctx context.Context, workspaceID string, _ *VariableListOptions) (*VariableList, error) {
	resp, err := c.client.API.Workspaces().ByWorkspace_id(workspaceID).Vars().Get(ctx, nil)
	if err != nil {
		return nil, err
	}

	list := &VariableList{}
	if resp == nil {
		return list, nil
	}
	for _, v := range resp.GetData() {
		list.Items = append(list.Items, toVariable(v))
	}
	return list, nil
}

func (c *ClientWrapper) CreateVariable(ctx context.Context, workspaceID string, opts VariableCreateOptions) (*Variable, error) {
	attrs := models.NewVars_attributes()
	attrs.SetKey(opts.Key)
	attrs.SetValue(opts.Value)
	attrs.SetDescription(opts.Description)
	attrs.SetHcl(opts.HCL)
	attrs.SetSensitive(opts.Sensitive)
	if opts.Category != nil {
		category, err := parseVarCategory(*opts.Category)
		if err != nil {
			return nil, err
		}
		attrs.SetCategory(category)
	}

	resp, err := c.client.API.Workspaces().ByWorkspace_id(workspaceID).Vars().Post(ctx, newVarsEnvelope("", attrs), nil)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.GetData() == nil {
		return nil, fmt.Errorf("create variable: empty response")
	}
	return toVariable(resp.GetData()), nil
}

func (c *ClientWrapper) UpdateVariable(ctx context.Context, workspaceID string, variableID string, opts VariableUpdateOptions) (*Variable, error) {
	attrs := models.NewVars_attributes()
	attrs.SetKey(opts.Key)
	attrs.SetValue(opts.Value)
	attrs.SetDescription(opts.Description)
	attrs.SetHcl(opts.HCL)
	attrs.SetSensitive(opts.Sensitive)

	resp, err := c.client.API.Workspaces().ByWorkspace_id(workspaceID).Vars().ById(variableID).Patch(ctx, newVarsEnvelope(variableID, attrs), nil)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.GetData() == nil {
		return nil, fmt.Errorf("update variable: empty response")
	}
	return toVariable(resp.GetData()), nil
}

func (c *ClientWrapper) DeleteVariable(ctx context.Context, workspaceID string, variableID string) error {
	return c.client.API.Workspaces().ByWorkspace_id(workspaceID).Vars().ById(variableID).Delete(ctx, nil)
}

func (c *ClientWrapper) ListProjects(ctx context.Context, org string, opts *ProjectListOptions) (*ProjectList, error) {
	params := &organizations.ItemProjectsRequestBuilderGetQueryParameters{}
	if opts != nil {
		params.Pagenumber = int32Ptr(opts.PageNumber)
		params.Pagesize = int32Ptr(opts.PageSize)
		params.Filternames = stringPtr(opts.Name)
	}

	resp, err := c.client.API.Organizations().ByOrganization_name(org).Projects().Get(ctx, &abs.RequestConfiguration[organizations.ItemProjectsRequestBuilderGetQueryParameters]{
		QueryParameters: params,
	})
	if err != nil {
		return nil, err
	}

	list := &ProjectList{}
	if resp == nil {
		return list, nil
	}
	if meta := resp.GetMeta(); meta != nil {
		list.Pagination = toPagination(meta.GetPagination())
	}
	for _, p := range resp.GetData() {
		list.Items = append(list.Items, toProject(p))
	}
	return list, nil
}

// --- conversion helpers from go-tfe v2 models ---

func toPagination(p models.Paginationable) *Pagination {
	if p == nil {
		return nil
	}
	return &Pagination{
		CurrentPage: int(deref(p.GetCurrentPage())),
		NextPage:    int(deref(p.GetNextPage())),
		TotalPages:  int(deref(p.GetTotalPages())),
		TotalCount:  int(deref(p.GetTotalCount())),
	}
}

func toOrganization(o models.Organizationsable) *Organization {
	org := &Organization{}
	if a := o.GetAttributes(); a != nil {
		org.Name = deref(a.GetName())
		org.Email = deref(a.GetEmail())
		org.CreatedAt = deref(a.GetCreatedAt())
	}
	return org
}

func toWorkspace(w models.Workspacesable) *Workspace {
	ws := &Workspace{ID: deref(w.GetId())}
	if a := w.GetAttributes(); a != nil {
		ws.Name = deref(a.GetName())
		ws.Description = deref(a.GetDescription())
		ws.ExecutionMode = enumString(a.GetExecutionMode())
		ws.TerraformVersion = deref(a.GetTerraformVersion())
		ws.Locked = deref(a.GetLocked())
		ws.AutoApply = deref(a.GetAutoApply())
		ws.WorkingDirectory = deref(a.GetWorkingDirectory())
		ws.ResourceCount = int(deref(a.GetResourceCount()))
		ws.CreatedAt = deref(a.GetCreatedAt())
		ws.UpdatedAt = deref(a.GetUpdatedAt())
	}
	return ws
}

// toRun converts a run model. included holds the side-loaded resources of a
// run read request and is used to fill in plan details; it may be nil.
func toRun(r models.Runsable, included []models.RunsEnvelope_RunsEnvelope_includedable) *Run {
	run := &Run{ID: deref(r.GetId())}
	if a := r.GetAttributes(); a != nil {
		run.Status = RunStatus(enumString(a.GetStatus()))
		run.Message = deref(a.GetMessage())
		run.TerraformVersion = deref(a.GetTerraformVersion())
		run.PlanOnly = deref(a.GetPlanOnly())
		run.HasChanges = deref(a.GetHasChanges())
		run.CreatedAt = deref(a.GetCreatedAt())
		if st := a.GetStatusTimestamps(); st != nil {
			run.StatusTimestamps = &RunStatusTimestamps{
				PlannedAt: deref(st.GetPlannedAt()),
				AppliedAt: deref(st.GetAppliedAt()),
			}
		}
	}

	rel := r.GetRelationships()
	if rel == nil {
		return run
	}

	if planRel := rel.GetPlan(); planRel != nil && planRel.GetData() != nil {
		planID := planRel.GetData().GetId()
		run.Plan = &Plan{ID: deref(planID)}
		plan := tfe.FindSideloadedResource(planID, included, func(item models.RunsEnvelope_RunsEnvelope_includedable) models.Plansable {
			return item.GetPlans()
		})
		if plan != nil && plan.GetAttributes() != nil {
			pa := plan.GetAttributes()
			run.Plan.ResourceAdditions = int(deref(pa.GetResourceAdditions()))
			run.Plan.ResourceChanges = int(deref(pa.GetResourceChanges()))
			run.Plan.ResourceDestructions = int(deref(pa.GetResourceDestructions()))
		}
	}

	if applyRel := rel.GetApply(); applyRel != nil && applyRel.GetData() != nil {
		run.Apply = &Apply{ID: deref(applyRel.GetData().GetId())}
	}

	return run
}

func toVariable(v models.Varsable) *Variable {
	variable := &Variable{ID: deref(v.GetId())}
	if a := v.GetAttributes(); a != nil {
		variable.Key = deref(a.GetKey())
		variable.Value = deref(a.GetValue())
		variable.Description = deref(a.GetDescription())
		variable.Category = CategoryType(enumString(a.GetCategory()))
		variable.HCL = deref(a.GetHcl())
		variable.Sensitive = deref(a.GetSensitive())
	}
	return variable
}

func toProject(p models.Projectsable) *Project {
	project := &Project{ID: deref(p.GetId())}
	if a := p.GetAttributes(); a != nil {
		project.Name = deref(a.GetName())
		project.Description = deref(a.GetDescription())
		project.DefaultExecutionMode = enumString(a.GetDefaultExecutionMode())
	}
	return project
}

func newVarsEnvelope(id string, attrs models.Vars_attributesable) *models.VarsEnvelope {
	varsType := models.VARS_VARS_TYPE
	data := models.NewVars()
	data.SetTypeEscaped(&varsType)
	data.SetAttributes(attrs)
	if id != "" {
		// The generated model treats "id" as read-only and never serializes it,
		// so it is passed as additional data for update requests.
		data.GetAdditionalData()["id"] = id
	}

	body := models.NewVarsEnvelope()
	body.SetData(data)
	return body
}

func parseVarCategory(category CategoryType) (*models.Vars_attributes_category, error) {
	parsed, err := models.ParseVars_attributes_category(string(category))
	if err != nil {
		return nil, fmt.Errorf("invalid variable category %q: %w", category, err)
	}
	c, ok := parsed.(*models.Vars_attributes_category)
	if !ok || c == nil {
		return nil, fmt.Errorf("invalid variable category %q", category)
	}
	return c, nil
}

// deref returns the value p points to, or the zero value if p is nil.
func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// enumString returns the API string of a Kiota enum, or "" if p is nil.
func enumString[T fmt.Stringer](p *T) string {
	if p == nil {
		return ""
	}
	return (*p).String()
}

// stringPtr returns a pointer to s, or nil if s is empty so the query
// parameter or attribute is omitted from the request.
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// int32Ptr returns a pointer to n as int32, or nil if n is not positive so
// the query parameter is omitted from the request.
func int32Ptr(n int) *int32 {
	if n <= 0 {
		return nil
	}
	v := int32(min(n, math.MaxInt32)) //nolint:gosec // G115: n is clamped to the int32 range
	return &v
}
