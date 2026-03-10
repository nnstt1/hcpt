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
	Before          interface{} `json:"before"`
	After           interface{} `json:"after"`
	KnownAfterApply bool        `json:"known_after_apply,omitempty"`
	Sensitive       bool        `json:"sensitive,omitempty"`
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
			diffs := computeDiffs(r.Before, r.After, r.AfterUnknown, r.BeforeSensitive, r.AfterSensitive)
			if len(diffs) > 0 {
				rj.Changes = make(map[string]driftChange, len(diffs))
				for _, d := range diffs {
					rj.Changes[d.Key] = driftChange{Before: d.BeforeRaw, After: d.AfterRaw, KnownAfterApply: d.KnownAfterApply, Sensitive: d.Sensitive}
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
	Key             string
	Before          string
	After           string
	BeforeRaw       interface{}
	AfterRaw        interface{}
	KnownAfterApply bool
	Sensitive       bool
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

// isKnownAfterApply checks if a flattened key (e.g. "a.b.c") is marked as unknown
// in flatAfterUnknown. It checks the key itself and all ancestor keys, because
// after_unknown may mark an entire parent object as true (e.g. "a": true) rather
// than listing each child individually.
func isKnownAfterApply(key string, flatAfterUnknown map[string]interface{}) bool {
	if flatAfterUnknown[key] == true {
		return true
	}
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			if flatAfterUnknown[key[:i]] == true {
				return true
			}
		}
	}
	return false
}

// computeDiffs compares before and after maps and returns sorted attribute diffs.
// afterUnknown marks attributes whose after value will be known only after apply.
// beforeSensitive/afterSensitive mark attributes whose values must not be displayed.
func computeDiffs(before, after, afterUnknown, beforeSensitive, afterSensitive map[string]interface{}) []attributeDiff {
	flatBefore := make(map[string]interface{})
	flatAfter := make(map[string]interface{})
	flatAfterUnknown := make(map[string]interface{})
	flatBeforeSensitive := make(map[string]interface{})
	flatAfterSensitive := make(map[string]interface{})

	if before != nil {
		flattenMap("", before, flatBefore)
	}
	if after != nil {
		flattenMap("", after, flatAfter)
	}
	if afterUnknown != nil {
		flattenMap("", afterUnknown, flatAfterUnknown)
	}
	if beforeSensitive != nil {
		flattenMap("", beforeSensitive, flatBeforeSensitive)
	}
	if afterSensitive != nil {
		flattenMap("", afterSensitive, flatAfterSensitive)
	}

	// Collect all keys from before and after
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
		unknown := isKnownAfterApply(k, flatAfterUnknown)
		sensitive := isKnownAfterApply(k, flatBeforeSensitive) || isKnownAfterApply(k, flatAfterSensitive)

		// Raw strings for change detection; display strings for output
		rawBStr := formatDiffValue(bVal)
		rawAStr := formatDiffValue(aVal)
		dispBStr := rawBStr
		if sensitive {
			dispBStr = "(sensitive value)"
		}

		if !bOk {
			// Added
			if aVal == nil && !unknown {
				continue
			}
			dispAStr := "(known after apply)"
			if !unknown {
				dispAStr = rawAStr
				if sensitive {
					dispAStr = "(sensitive value)"
				}
			}
			diffs = append(diffs, attributeDiff{Key: k, Before: "(null)", After: dispAStr, BeforeRaw: nil, AfterRaw: aVal, KnownAfterApply: unknown, Sensitive: sensitive})
		} else if !aOk || unknown {
			// Removed or known-after-apply (key absent from after, or parent marked unknown)
			if bVal == nil && !unknown {
				continue
			}
			if unknown {
				diffs = append(diffs, attributeDiff{Key: k, Before: dispBStr, After: "(known after apply)", BeforeRaw: bVal, AfterRaw: nil, KnownAfterApply: true, Sensitive: sensitive})
			} else {
				diffs = append(diffs, attributeDiff{Key: k, Before: dispBStr, After: "(null)", BeforeRaw: bVal, AfterRaw: nil, Sensitive: sensitive})
			}
		} else if rawBStr != rawAStr {
			// Changed — compare raw values; display masked if sensitive
			dispAStr := rawAStr
			if sensitive {
				dispAStr = "(sensitive value)"
			}
			diffs = append(diffs, attributeDiff{Key: k, Before: dispBStr, After: dispAStr, BeforeRaw: bVal, AfterRaw: aVal, Sensitive: sensitive})
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
		diffs := computeDiffs(r.Before, r.After, r.AfterUnknown, r.BeforeSensitive, r.AfterSensitive)
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
