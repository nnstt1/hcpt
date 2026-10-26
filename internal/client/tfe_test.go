package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe/v2"
)

// newTestSDKClientWrapper creates a ClientWrapper backed by a real go-tfe v2
// client that points to the given test server.
func newTestSDKClientWrapper(t *testing.T, serverURL string) *ClientWrapper {
	t.Helper()
	c, err := tfe.NewClient(&tfe.Config{Address: serverURL, Token: "test-token"})
	if err != nil {
		t.Fatalf("failed to create tfe client: %v", err)
	}
	return &ClientWrapper{client: c, address: serverURL, token: "test-token"}
}

// jsonAPIHandler returns a handler that checks the request path and responds with body.
func jsonAPIHandler(t *testing.T, wantPath string, body string, check func(r *http.Request)) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("expected path %q, got %q", wantPath, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("expected bearer token, got %q", got)
		}
		if check != nil {
			check(r)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprint(w, body)
	}
}

func TestListOrganizations_SDK(t *testing.T) {
	body := `{
		"data": [
			{"id": "org-a", "type": "organizations", "attributes": {"name": "org-a", "email": "a@example.com", "created-at": "2024-01-02T03:04:05Z"}}
		],
		"meta": {"pagination": {"current-page": 1, "next-page": 2, "total-pages": 2, "total-count": 2}}
	}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/organizations", body, func(r *http.Request) {
		q := r.URL.Query()
		if q.Get("page[size]") != "100" || q.Get("page[number]") != "2" {
			t.Errorf("unexpected pagination query: %v", q)
		}
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	list, err := cw.ListOrganizations(context.Background(), &OrganizationListOptions{ListOptions: ListOptions{PageNumber: 2, PageSize: 100}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(list.Items))
	}
	org := list.Items[0]
	if org.Name != "org-a" || org.Email != "a@example.com" {
		t.Errorf("unexpected organization: %+v", org)
	}
	if !org.CreatedAt.Equal(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("unexpected CreatedAt: %v", org.CreatedAt)
	}
	if list.Pagination == nil || list.NextPage != 2 || list.TotalPages != 2 {
		t.Errorf("unexpected pagination: %+v", list.Pagination)
	}
}

func TestReadOrganization_SDK(t *testing.T) {
	body := `{"data": {"id": "my-org", "type": "organizations", "attributes": {"name": "my-org", "email": "me@example.com"}}}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/organizations/my-org", body, nil))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	org, err := cw.ReadOrganization(context.Background(), "my-org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.Name != "my-org" || org.Email != "me@example.com" {
		t.Errorf("unexpected organization: %+v", org)
	}
}

func TestReadOrganization_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"errors": [{"status": "404", "title": "not found"}]}`)
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	_, err := cw.ReadOrganization(context.Background(), "missing")
	if !errors.Is(err, tfe.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestReadEntitlements(t *testing.T) {
	body := `{"data": {"id": "org-x", "type": "entitlement-sets", "attributes": {
		"agents": true, "audit-logging": true, "cost-estimation": false, "operations": true,
		"private-module-registry": true, "run-tasks": false, "sso": true, "sentinel": false,
		"state-storage": true, "teams": true, "vcs-integrations": true
	}}}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/organizations/my-org/entitlement-set", body, nil))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	ent, err := cw.ReadEntitlements(context.Background(), "my-org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Entitlements{
		Agents: true, AuditLogging: true, Operations: true, PrivateModuleRegistry: true,
		SSO: true, StateStorage: true, Teams: true, VCSIntegrations: true,
	}
	if *ent != want {
		t.Errorf("expected %+v, got %+v", want, *ent)
	}
}

func TestReadEntitlements_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	if _, err := cw.ReadEntitlements(context.Background(), "my-org"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListWorkspaces_SDK(t *testing.T) {
	body := `{
		"data": [{"id": "ws-1", "type": "workspaces", "attributes": {"name": "ws-one"}}],
		"meta": {"pagination": {"current-page": 1, "next-page": null, "total-pages": 1, "total-count": 1}}
	}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/organizations/my-org/workspaces", body, func(r *http.Request) {
		if got := r.URL.Query().Get("filter[project][id]"); got != "prj-1" {
			t.Errorf("expected project filter prj-1, got %q", got)
		}
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	list, err := cw.ListWorkspaces(context.Background(), "my-org", &WorkspaceListOptions{ProjectID: "prj-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != "ws-1" || list.Items[0].Name != "ws-one" {
		t.Errorf("unexpected items: %+v", list.Items)
	}
	if list.Pagination == nil || list.NextPage != 0 {
		t.Errorf("unexpected pagination: %+v", list.Pagination)
	}
}

func TestReadWorkspace_SDK(t *testing.T) {
	body := `{"data": {"id": "ws-abc", "type": "workspaces", "attributes": {
		"name": "my-ws", "description": "desc", "execution-mode": "agent",
		"terraform-version": "1.9.0", "locked": true, "auto-apply": true,
		"working-directory": "infra", "resource-count": 7,
		"created-at": "2024-01-01T00:00:00Z", "updated-at": "2024-02-01T00:00:00Z"
	}}}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/organizations/my-org/workspaces/my-ws", body, nil))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	ws, err := cw.ReadWorkspace(context.Background(), "my-org", "my-ws")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Workspace{
		ID: "ws-abc", Name: "my-ws", Description: "desc", ExecutionMode: "agent",
		TerraformVersion: "1.9.0", Locked: true, AutoApply: true, WorkingDirectory: "infra",
		ResourceCount: 7,
		CreatedAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	if ws.ID != want.ID || ws.Name != want.Name || ws.Description != want.Description ||
		ws.ExecutionMode != want.ExecutionMode || ws.TerraformVersion != want.TerraformVersion ||
		ws.Locked != want.Locked || ws.AutoApply != want.AutoApply ||
		ws.WorkingDirectory != want.WorkingDirectory || ws.ResourceCount != want.ResourceCount ||
		!ws.CreatedAt.Equal(want.CreatedAt) || !ws.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("expected %+v, got %+v", want, *ws)
	}
}

func TestListRuns_SDK(t *testing.T) {
	body := `{
		"data": [{"id": "run-1", "type": "runs", "attributes": {
			"status": "planned_and_finished", "message": "hello", "plan-only": true,
			"has-changes": false, "created-at": "2024-05-01T00:00:00Z"
		}}],
		"meta": {"pagination": {"current-page": 1, "next-page": null, "total-pages": 1, "total-count": 1}}
	}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/workspaces/ws-abc/runs", body, func(r *http.Request) {
		q := r.URL.Query()
		if q.Get("filter[operation]") != "plan_only,destroy" {
			t.Errorf("unexpected operation filter: %q", q.Get("filter[operation]"))
		}
		if q.Get("filter[status]") != "applied,errored" {
			t.Errorf("unexpected status filter: %q", q.Get("filter[status]"))
		}
		if q.Get("page[size]") != "1" {
			t.Errorf("unexpected page size: %q", q.Get("page[size]"))
		}
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	list, err := cw.ListRuns(context.Background(), "ws-abc", &RunListOptions{
		ListOptions: ListOptions{PageSize: 1},
		Operation:   "plan_only,destroy",
		Status:      "applied,errored",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 run, got %d", len(list.Items))
	}
	r := list.Items[0]
	if r.ID != "run-1" || r.Status != RunPlannedAndFinished || r.Message != "hello" || !r.PlanOnly || r.HasChanges {
		t.Errorf("unexpected run: %+v", r)
	}
}

func TestReadRunWithApply_SDK(t *testing.T) {
	body := `{
		"data": {
			"id": "run-xyz", "type": "runs",
			"attributes": {
				"status": "applied", "message": "deploy", "terraform-version": "1.9.0",
				"has-changes": true, "created-at": "2024-05-01T00:00:00Z",
				"status-timestamps": {"planned-at": "2024-05-01T00:01:00Z", "applied-at": "2024-05-01T00:02:00Z"}
			},
			"relationships": {
				"plan": {"data": {"id": "plan-1", "type": "plans"}},
				"apply": {"data": {"id": "apply-1", "type": "applies"}}
			}
		},
		"included": [
			{"id": "plan-1", "type": "plans", "attributes": {"resource-additions": 1, "resource-changes": 2, "resource-destructions": 3}},
			{"id": "apply-1", "type": "applies", "attributes": {}}
		]
	}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/runs/run-xyz", body, func(r *http.Request) {
		if got := r.URL.Query().Get("include"); got != "plan,apply" {
			t.Errorf("expected include=plan,apply, got %q", got)
		}
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	r, err := cw.ReadRunWithApply(context.Background(), "run-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ID != "run-xyz" || r.Status != RunApplied || r.Message != "deploy" || r.TerraformVersion != "1.9.0" || !r.HasChanges {
		t.Errorf("unexpected run: %+v", r)
	}
	if r.StatusTimestamps == nil ||
		!r.StatusTimestamps.PlannedAt.Equal(time.Date(2024, 5, 1, 0, 1, 0, 0, time.UTC)) ||
		!r.StatusTimestamps.AppliedAt.Equal(time.Date(2024, 5, 1, 0, 2, 0, 0, time.UTC)) {
		t.Errorf("unexpected status timestamps: %+v", r.StatusTimestamps)
	}
	if r.Plan == nil || *r.Plan != (Plan{ID: "plan-1", ResourceAdditions: 1, ResourceChanges: 2, ResourceDestructions: 3}) {
		t.Errorf("unexpected plan: %+v", r.Plan)
	}
	if r.Apply == nil || r.Apply.ID != "apply-1" {
		t.Errorf("unexpected apply: %+v", r.Apply)
	}
}

func TestReadRun_NoPlanIncluded(t *testing.T) {
	body := `{"data": {"id": "run-1", "type": "runs", "attributes": {"status": "pending"},
		"relationships": {"plan": {"data": {"id": "plan-1", "type": "plans"}}}}}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/runs/run-1", body, func(r *http.Request) {
		if got := r.URL.Query().Get("include"); got != "plan" {
			t.Errorf("expected include=plan, got %q", got)
		}
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	r, err := cw.ReadRun(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status != RunPending {
		t.Errorf("expected pending, got %q", r.Status)
	}
	if r.Plan == nil || r.Plan.ID != "plan-1" || r.Plan.ResourceAdditions != 0 {
		t.Errorf("unexpected plan: %+v", r.Plan)
	}
	if r.Apply != nil {
		t.Errorf("expected nil apply, got %+v", r.Apply)
	}
}

func TestReadPlanJSONOutput_FollowsRedirect(t *testing.T) {
	planJSON := `{"resource_changes": []}`
	archivist := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("expected Authorization header to be dropped on cross-origin redirect")
		}
		_, _ = fmt.Fprint(w, planJSON)
	}))
	defer archivist.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/plans/plan-1/json-output" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		http.Redirect(w, r, archivist.URL+"/v1/object/abc", http.StatusTemporaryRedirect)
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	got, err := cw.ReadPlanJSONOutput(context.Background(), "plan-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != planJSON {
		t.Errorf("expected %q, got %q", planJSON, string(got))
	}
}

func TestReadPlanJSONOutput_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	if _, err := cw.ReadPlanJSONOutput(context.Background(), "plan-1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReadApplyLogs_SDK(t *testing.T) {
	logs := "line1\nline2\n"
	archivist := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, logs)
	}))
	defer archivist.Close()

	body := fmt.Sprintf(`{"data": {"id": "apply-1", "type": "applies", "attributes": {"log-read-url": %q}}}`, archivist.URL+"/v1/object/logs")
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/applies/apply-1", body, nil))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	r, err := cw.ReadApplyLogs(context.Background(), "apply-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := io.ReadAll(r)
	if string(got) != logs {
		t.Errorf("expected %q, got %q", logs, string(got))
	}
}

func TestReadApplyLogs_NoLogURL(t *testing.T) {
	body := `{"data": {"id": "apply-1", "type": "applies", "attributes": {}}}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/applies/apply-1", body, nil))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	if _, err := cw.ReadApplyLogs(context.Background(), "apply-1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListVariables_SDK(t *testing.T) {
	body := `{"data": [
		{"id": "var-1", "type": "vars", "attributes": {"key": "region", "value": "japaneast", "category": "terraform", "hcl": false, "sensitive": false, "description": "d"}},
		{"id": "var-2", "type": "vars", "attributes": {"key": "TOKEN", "value": null, "category": "env", "hcl": false, "sensitive": true}}
	]}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/workspaces/ws-abc/vars", body, nil))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	list, err := cw.ListVariables(context.Background(), "ws-abc", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(list.Items))
	}
	if *list.Items[0] != (Variable{ID: "var-1", Key: "region", Value: "japaneast", Description: "d", Category: CategoryTerraform}) {
		t.Errorf("unexpected variable: %+v", list.Items[0])
	}
	if *list.Items[1] != (Variable{ID: "var-2", Key: "TOKEN", Category: CategoryEnv, Sensitive: true}) {
		t.Errorf("unexpected variable: %+v", list.Items[1])
	}
}

// decodeVarRequest decodes a vars create/update request body.
func decodeVarRequest(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var req struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return req.Data
}

func TestCreateVariable_SDK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/workspaces/ws-abc/vars" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		data := decodeVarRequest(t, r)
		if data["type"] != "vars" {
			t.Errorf("expected type vars, got %v", data["type"])
		}
		if _, ok := data["id"]; ok {
			t.Errorf("expected no id on create, got %v", data["id"])
		}
		attrs, _ := data["attributes"].(map[string]any)
		want := map[string]any{"key": "region", "value": "japaneast", "category": "env", "hcl": true, "sensitive": false, "description": ""}
		for k, v := range want {
			if attrs[k] != v {
				t.Errorf("attribute %q: expected %v, got %v", k, v, attrs[k])
			}
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"data": {"id": "var-new", "type": "vars", "attributes": {"key": "region", "value": "japaneast", "category": "env", "hcl": true}}}`)
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	key, value, desc := "region", "japaneast", ""
	category := CategoryEnv
	hcl, sensitive := true, false
	v, err := cw.CreateVariable(context.Background(), "ws-abc", VariableCreateOptions{
		Key: &key, Value: &value, Description: &desc, Category: &category, HCL: &hcl, Sensitive: &sensitive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != "var-new" || v.Category != CategoryEnv || !v.HCL {
		t.Errorf("unexpected variable: %+v", v)
	}
}

func TestCreateVariable_InvalidCategory(t *testing.T) {
	cw := newTestSDKClientWrapper(t, "http://127.0.0.1:0")
	category := CategoryType("invalid")
	if _, err := cw.CreateVariable(context.Background(), "ws-abc", VariableCreateOptions{Category: &category}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateVariable_SDK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/workspaces/ws-abc/vars/var-1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		data := decodeVarRequest(t, r)
		if data["id"] != "var-1" {
			t.Errorf("expected id var-1, got %v", data["id"])
		}
		attrs, _ := data["attributes"].(map[string]any)
		if attrs["value"] != "new" {
			t.Errorf("expected value new, got %v", attrs["value"])
		}
		if _, ok := attrs["category"]; ok {
			t.Error("expected category to be omitted on update")
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprint(w, `{"data": {"id": "var-1", "type": "vars", "attributes": {"key": "region", "value": "new", "category": "terraform"}}}`)
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	value := "new"
	v, err := cw.UpdateVariable(context.Background(), "ws-abc", "var-1", VariableUpdateOptions{Value: &value})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Value != "new" {
		t.Errorf("unexpected variable: %+v", v)
	}
}

func TestDeleteVariable_SDK(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/workspaces/ws-abc/vars/var-1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	if err := cw.DeleteVariable(context.Background(), "ws-abc", "var-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected delete request to be sent")
	}
}

func TestListProjects_SDK(t *testing.T) {
	body := `{
		"data": [{"id": "prj-1", "type": "projects", "attributes": {"name": "my-project", "description": "d", "default-execution-mode": "remote"}}],
		"meta": {"pagination": {"current-page": 1, "next-page": null, "total-pages": 1, "total-count": 1}}
	}`
	ts := httptest.NewServer(jsonAPIHandler(t, "/api/v2/organizations/my-org/projects", body, func(r *http.Request) {
		if got := r.URL.Query().Get("filter[names]"); got != "my-project" {
			t.Errorf("expected filter[names]=my-project, got %q", got)
		}
	}))
	defer ts.Close()

	cw := newTestSDKClientWrapper(t, ts.URL)
	list, err := cw.ListProjects(context.Background(), "my-org", &ProjectListOptions{Name: "my-project"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 1 || *list.Items[0] != (Project{ID: "prj-1", Name: "my-project", Description: "d", DefaultExecutionMode: "remote"}) {
		t.Errorf("unexpected projects: %+v", list.Items)
	}
}

func TestInt32Ptr(t *testing.T) {
	if int32Ptr(0) != nil || int32Ptr(-1) != nil {
		t.Error("expected nil for non-positive values")
	}
	if got := int32Ptr(100); got == nil || *got != 100 {
		t.Errorf("expected 100, got %v", got)
	}
}
