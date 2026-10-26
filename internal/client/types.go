package client

import "time"

// The types below are hcpt's own representation of HCP Terraform resources.
// go-tfe v2 exposes Kiota-generated models (interfaces with pointer getters),
// so ClientWrapper converts them into these plain structs to keep the command
// layer and its test mocks independent of the SDK.

// ListOptions holds pagination options for list requests.
type ListOptions struct {
	PageNumber int
	PageSize   int
}

// Pagination holds pagination metadata returned by list endpoints.
type Pagination struct {
	CurrentPage int
	NextPage    int
	TotalPages  int
	TotalCount  int
}

// Organization represents an HCP Terraform organization.
type Organization struct {
	Name      string
	Email     string
	CreatedAt time.Time
}

// OrganizationListOptions holds options for listing organizations.
type OrganizationListOptions struct {
	ListOptions
}

// OrganizationList holds a page of organizations.
type OrganizationList struct {
	*Pagination
	Items []*Organization
}

// Entitlements represents the features an organization is entitled to.
type Entitlements struct {
	Agents                bool
	AuditLogging          bool
	CostEstimation        bool
	Operations            bool
	PrivateModuleRegistry bool
	RunTasks              bool
	SSO                   bool
	Sentinel              bool
	StateStorage          bool
	Teams                 bool
	VCSIntegrations       bool
}

// Workspace represents an HCP Terraform workspace.
type Workspace struct {
	ID               string
	Name             string
	Description      string
	ExecutionMode    string
	TerraformVersion string
	Locked           bool
	AutoApply        bool
	WorkingDirectory string
	ResourceCount    int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// WorkspaceListOptions holds options for listing workspaces.
type WorkspaceListOptions struct {
	ListOptions
	ProjectID string
}

// WorkspaceList holds a page of workspaces.
type WorkspaceList struct {
	*Pagination
	Items []*Workspace
}

// RunStatus represents the status of a run.
type RunStatus string

// Run statuses referenced by hcpt.
const (
	RunPending            RunStatus = "pending"
	RunPlanning           RunStatus = "planning"
	RunPlanned            RunStatus = "planned"
	RunApplying           RunStatus = "applying"
	RunApplied            RunStatus = "applied"
	RunErrored            RunStatus = "errored"
	RunCanceled           RunStatus = "canceled"
	RunDiscarded          RunStatus = "discarded"
	RunPlannedAndFinished RunStatus = "planned_and_finished"
	RunPlannedAndSaved    RunStatus = "planned_and_saved"
)

// Run represents an HCP Terraform run.
type Run struct {
	ID               string
	Status           RunStatus
	Message          string
	TerraformVersion string
	PlanOnly         bool
	HasChanges       bool
	CreatedAt        time.Time
	StatusTimestamps *RunStatusTimestamps
	Plan             *Plan
	Apply            *Apply
}

// RunStatusTimestamps holds the timestamps of run status transitions.
type RunStatusTimestamps struct {
	PlannedAt time.Time
	AppliedAt time.Time
}

// Plan represents the plan of a run.
type Plan struct {
	ID                   string
	ResourceAdditions    int
	ResourceChanges      int
	ResourceDestructions int
}

// Apply represents the apply of a run.
type Apply struct {
	ID string
}

// RunListOptions holds options for listing runs.
type RunListOptions struct {
	ListOptions
	// Operation is a comma-separated list of run operations to include.
	Operation string
	// Status is a comma-separated list of run statuses to include.
	Status string
}

// RunList holds a page of runs.
type RunList struct {
	*Pagination
	Items []*Run
}

// CategoryType represents the category of a variable.
type CategoryType string

// Variable categories.
const (
	CategoryTerraform CategoryType = "terraform"
	CategoryEnv       CategoryType = "env"
)

// Variable represents a workspace variable.
type Variable struct {
	ID          string
	Key         string
	Value       string
	Description string
	Category    CategoryType
	HCL         bool
	Sensitive   bool
}

// VariableListOptions holds options for listing variables.
type VariableListOptions struct {
	ListOptions
}

// VariableList holds the variables of a workspace.
type VariableList struct {
	*Pagination
	Items []*Variable
}

// VariableCreateOptions holds options for creating a variable.
type VariableCreateOptions struct {
	Key         *string
	Value       *string
	Description *string
	Category    *CategoryType
	HCL         *bool
	Sensitive   *bool
}

// VariableUpdateOptions holds options for updating a variable.
type VariableUpdateOptions struct {
	Key         *string
	Value       *string
	Description *string
	HCL         *bool
	Sensitive   *bool
}

// Project represents an HCP Terraform project.
type Project struct {
	ID                   string
	Name                 string
	Description          string
	DefaultExecutionMode string
}

// ProjectListOptions holds options for listing projects.
type ProjectListOptions struct {
	ListOptions
	// Name filters projects by name.
	Name string
}

// ProjectList holds a page of projects.
type ProjectList struct {
	*Pagination
	Items []*Project
}
