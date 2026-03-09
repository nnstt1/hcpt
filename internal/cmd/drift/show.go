package drift

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type driftShowService interface {
	client.WorkspaceService
	client.AssessmentService
}

type driftShowClientFactory func() (driftShowService, error)

func defaultDriftShowClientFactory() (driftShowService, error) {
	return client.NewClientWrapper()
}

func newCmdDriftShow() *cobra.Command {
	return newCmdDriftShowWith(defaultDriftShowClientFactory)
}

func newCmdDriftShowWith(clientFn driftShowClientFactory) *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:          "show <workspace>",
		Short:        "Show drift detection detail for a workspace",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return errOrgRequired
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runDriftShow(svc, org, args[0], verbose)
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show attribute-level diffs for drifted resources")
	return cmd
}

type driftResourceJSON struct {
	Address string                 `json:"address"`
	Type    string                 `json:"type"`
	Name    string                 `json:"name"`
	Action  string                 `json:"action"`
	Changes map[string]driftChange `json:"changes,omitempty"`
}

type driftChange struct {
	Before interface{} `json:"before"`
	After  interface{} `json:"after"`
}

type driftShowJSON struct {
	Workspace          string              `json:"workspace"`
	Drifted            *bool               `json:"drifted"`
	ResourcesDrifted   *int                `json:"resources_drifted"`
	ResourcesUndrifted *int                `json:"resources_undrifted"`
	LastAssessment     string              `json:"last_assessment"`
	DriftedResources   []driftResourceJSON `json:"drifted_resources,omitempty"`
}

func runDriftShow(svc driftShowService, org, name string, verbose bool) error {
	ctx := context.Background()
	ws, err := svc.ReadWorkspace(ctx, org, name)
	if err != nil {
		return fmt.Errorf("failed to read workspace %q: %w", name, err)
	}

	result, err := svc.ReadCurrentAssessment(ctx, ws.ID)
	if err != nil {
		return fmt.Errorf("failed to read assessment for workspace %q: %w", name, err)
	}

	// Fetch drifted resource details if assessment shows drift
	var driftedResources []client.DriftedResource
	if result != nil && (result.Drifted || result.ResourcesDrifted > 0) && result.ID != "" {
		driftedResources, err = svc.ReadAssessmentDriftDetails(ctx, result.ID)
		if err != nil {
			// Non-fatal: show summary even if details fail
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch drift details: %v\n", err)
		}
	}

	if viper.GetBool("json") {
		return output.PrintJSON(os.Stdout, toDriftShowJSON(ws, result, driftedResources, verbose))
	}

	pairs := buildDriftShowKeyValues(ws, result)
	output.PrintKeyValue(os.Stdout, pairs)

	if len(driftedResources) > 0 {
		fmt.Fprintln(os.Stdout)
		headers := []string{"RESOURCE", "TYPE", "ACTION"}
		rows := make([][]string, 0, len(driftedResources))
		for _, r := range driftedResources {
			rows = append(rows, []string{r.Address, r.Type, r.Action})
		}
		output.Print(os.Stdout, headers, rows)

		if verbose {
			printResourceDiffs(os.Stdout, driftedResources)
		}
	}

	return nil
}

func toDriftShowJSON(ws *tfe.Workspace, result *client.AssessmentResult, resources []client.DriftedResource, verbose bool) driftShowJSON {
	d := driftShowJSON{
		Workspace: ws.Name,
	}
	if result != nil {
		d.Drifted = &result.Drifted
		d.ResourcesDrifted = &result.ResourcesDrifted
		d.ResourcesUndrifted = &result.ResourcesUndrifted
		d.LastAssessment = result.CreatedAt
	}
	for _, r := range resources {
		rj := driftResourceJSON{
			Address: r.Address,
			Type:    r.Type,
			Name:    r.Name,
			Action:  r.Action,
		}
		if verbose {
			diffs := computeDiffs(r.Before, r.After)
			if len(diffs) > 0 {
				rj.Changes = make(map[string]driftChange, len(diffs))
				for _, d := range diffs {
					rj.Changes[d.Key] = driftChange{Before: d.BeforeRaw, After: d.AfterRaw}
				}
			}
		}
		d.DriftedResources = append(d.DriftedResources, rj)
	}
	return d
}

func buildDriftShowKeyValues(ws *tfe.Workspace, result *client.AssessmentResult) []output.KeyValue {
	pairs := []output.KeyValue{
		{Key: "Workspace", Value: ws.Name},
	}

	if result != nil {
		pairs = append(pairs,
			output.KeyValue{Key: "Drifted", Value: strconv.FormatBool(result.Drifted)},
			output.KeyValue{Key: "Resources Drifted", Value: strconv.Itoa(result.ResourcesDrifted)},
			output.KeyValue{Key: "Resources Undrifted", Value: strconv.Itoa(result.ResourcesUndrifted)},
			output.KeyValue{Key: "Last Assessment", Value: result.CreatedAt},
		)
	} else {
		pairs = append(pairs,
			output.KeyValue{Key: "Drifted", Value: "not ready"},
			output.KeyValue{Key: "Resources Drifted", Value: "-"},
			output.KeyValue{Key: "Resources Undrifted", Value: "-"},
			output.KeyValue{Key: "Last Assessment", Value: "-"},
		)
	}

	return pairs
}

// attributeDiff represents a single attribute-level change between before and after states.
type attributeDiff struct {
	Key       string
	Before    string
	After     string
	BeforeRaw interface{}
	AfterRaw  interface{}
}

// flattenMap recursively flattens a nested map into dot-notation keys.
// Empty maps and empty arrays are stored as leaf values to distinguish them from nil.
func flattenMap(prefix string, value interface{}, result map[string]interface{}) {
	switch v := value.(type) {
	case map[string]interface{}:
		if len(v) == 0 && prefix != "" {
			result[prefix] = value
			return
		}
		for k, val := range v {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flattenMap(key, val, result)
		}
	case []interface{}:
		if len(v) == 0 && prefix != "" {
			result[prefix] = value
			return
		}
		for i, val := range v {
			key := fmt.Sprintf("%s.%d", prefix, i)
			flattenMap(key, val, result)
		}
	default:
		result[prefix] = value
	}
}

// computeDiffs compares before and after maps and returns sorted attribute diffs.
func computeDiffs(before, after map[string]interface{}) []attributeDiff {
	flatBefore := make(map[string]interface{})
	flatAfter := make(map[string]interface{})

	if before != nil {
		flattenMap("", before, flatBefore)
	}
	if after != nil {
		flattenMap("", after, flatAfter)
	}

	// Collect all keys
	keys := make(map[string]struct{})
	for k := range flatBefore {
		keys[k] = struct{}{}
	}
	for k := range flatAfter {
		keys[k] = struct{}{}
	}

	var diffs []attributeDiff
	for k := range keys {
		bVal, bOk := flatBefore[k]
		aVal, aOk := flatAfter[k]

		bStr := formatDiffValue(bVal)
		aStr := formatDiffValue(aVal)

		if !bOk {
			// Added (skip if value is nil — no real change)
			if aVal == nil {
				continue
			}
			diffs = append(diffs, attributeDiff{Key: k, Before: "(null)", After: aStr, BeforeRaw: nil, AfterRaw: aVal})
		} else if !aOk {
			// Removed (skip if value is nil — no real change)
			if bVal == nil {
				continue
			}
			diffs = append(diffs, attributeDiff{Key: k, Before: bStr, After: "(null)", BeforeRaw: bVal, AfterRaw: nil})
		} else if bStr != aStr {
			// Changed
			diffs = append(diffs, attributeDiff{Key: k, Before: bStr, After: aStr, BeforeRaw: bVal, AfterRaw: aVal})
		}
	}

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Key < diffs[j].Key
	})

	return diffs
}

// formatDiffValue converts a value to a display string.
func formatDiffValue(v interface{}) string {
	if v == nil {
		return "(null)"
	}
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		return strconv.FormatBool(val)
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		return fmt.Sprintf("%v", val)
	case map[string]interface{}:
		if len(val) == 0 {
			return "{}"
		}
		return fmt.Sprintf("%v", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// printResourceDiffs prints attribute-level diffs for each drifted resource.
func printResourceDiffs(w *os.File, resources []client.DriftedResource) {
	for _, r := range resources {
		diffs := computeDiffs(r.Before, r.After)
		if len(diffs) == 0 {
			continue
		}

		fmt.Fprintf(w, "\nResource: %s (%s)\n", r.Address, r.Action)
		// Find max key length for alignment
		maxKeyLen := 0
		for _, d := range diffs {
			if len(d.Key) > maxKeyLen {
				maxKeyLen = len(d.Key)
			}
		}

		for _, d := range diffs {
			var symbol string
			switch {
			case d.Before == "(null)":
				symbol = "+"
			case d.After == "(null)":
				symbol = "-"
			default:
				symbol = "~"
			}
			padding := strings.Repeat(" ", maxKeyLen-len(d.Key))
			fmt.Fprintf(w, "  %s %s:%s  %s => %s\n", symbol, d.Key, padding, d.Before, d.After)
		}
	}
}
