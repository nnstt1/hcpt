package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// Variable represents a parsed variable with its key, value, and type information.
type Variable struct {
	Key   string
	Value string
	IsHCL bool
}

// ParseFile parses a .tfvars or .tfvars.json file and returns a list of variables.
func ParseFile(filename string) ([]Variable, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".tfvars":
		return parseHCLFile(filename)
	case ".json":
		return parseJSONFile(filename)
	default:
		// Check for .tfvars.json pattern
		if strings.HasSuffix(strings.ToLower(filename), ".tfvars.json") {
			return parseJSONFile(filename)
		}
		return nil, fmt.Errorf("unsupported file format: %s (expected .tfvars, .tfvars.json, or .json)", ext)
	}
}

// parseHCLFile parses a HCL-formatted .tfvars file.
func parseHCLFile(filename string) ([]Variable, error) {
	parser := hclparse.NewParser()
	file, diag := parser.ParseHCLFile(filename)
	if diag.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %w", diag)
	}

	attrs, diag := file.Body.JustAttributes()
	if diag.HasErrors() {
		return nil, fmt.Errorf("failed to get attributes: %w", diag)
	}

	vars := make([]Variable, 0, len(attrs))
	for name, attr := range attrs {
		val, diag := attr.Expr.Value(nil)
		if diag.HasErrors() {
			return nil, fmt.Errorf("failed to evaluate expression for %q: %w", name, diag)
		}

		v := Variable{Key: name}
		switch {
		case val.Type() == cty.String:
			v.Value = val.AsString()
			v.IsHCL = false
		case val.Type() == cty.Number:
			bf := val.AsBigFloat()
			// Format as integer if possible, otherwise as float
			if bf.IsInt() {
				i, _ := bf.Int64()
				v.Value = strconv.FormatInt(i, 10)
			} else {
				f, _ := bf.Float64()
				v.Value = strconv.FormatFloat(f, 'f', -1, 64)
			}
			v.IsHCL = false
		case val.Type() == cty.Bool:
			v.Value = strconv.FormatBool(val.True())
			v.IsHCL = false
		default:
			// Complex type: extract HCL source code
			bytes := file.Bytes[attr.Expr.Range().Start.Byte:attr.Expr.Range().End.Byte]
			v.Value = string(bytes)
			v.IsHCL = true
		}
		vars = append(vars, v)
	}

	return vars, nil
}

// parseJSONFile parses a JSON-formatted .tfvars.json file.
func parseJSONFile(filename string) ([]Variable, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	vars := make([]Variable, 0, len(raw))
	for key, val := range raw {
		v := Variable{Key: key}

		switch t := val.(type) {
		case string:
			v.Value = t
			v.IsHCL = false
		case float64:
			// Format as integer if possible, otherwise as float
			if t == float64(int64(t)) {
				v.Value = strconv.FormatInt(int64(t), 10)
			} else {
				v.Value = strconv.FormatFloat(t, 'f', -1, 64)
			}
			v.IsHCL = false
		case bool:
			v.Value = strconv.FormatBool(t)
			v.IsHCL = false
		case []interface{}, map[string]interface{}:
			// Complex type: convert to JSON string and mark as HCL
			bytes, err := json.Marshal(t)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal complex type for variable %q: %w", key, err)
			}
			v.Value = string(bytes)
			v.IsHCL = true
		default:
			return nil, fmt.Errorf("unsupported type %T for variable %q", t, key)
		}
		vars = append(vars, v)
	}

	return vars, nil
}
