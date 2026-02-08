package workspace

import (
	"time"

	tfe "github.com/hashicorp/go-tfe"
)

type workspaceJSON struct {
	Name             string `json:"name"`
	ID               string `json:"id"`
	Description      string `json:"description"`
	ExecutionMode    string `json:"execution_mode"`
	TerraformVersion string `json:"terraform_version"`
	Locked           bool   `json:"locked"`
	AutoApply        bool   `json:"auto_apply"`
	WorkingDirectory string `json:"working_directory"`
	ResourceCount    int    `json:"resource_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func toWorkspaceJSON(ws *tfe.Workspace) workspaceJSON {
	return workspaceJSON{
		Name:             ws.Name,
		ID:               ws.ID,
		Description:      ws.Description,
		ExecutionMode:    ws.ExecutionMode,
		TerraformVersion: ws.TerraformVersion,
		Locked:           ws.Locked,
		AutoApply:        ws.AutoApply,
		WorkingDirectory: ws.WorkingDirectory,
		ResourceCount:    ws.ResourceCount,
		CreatedAt:        ws.CreatedAt,
		UpdatedAt:        ws.UpdatedAt,
	}
}
