package parser

import (
	"path/filepath"
	"testing"
)

func TestParseFile_SimpleHCL(t *testing.T) {
	vars, err := ParseFile(filepath.Join("testdata", "simple.tfvars"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(vars) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(vars))
	}

	// Check region
	found := false
	for _, v := range vars {
		if v.Key == "region" {
			found = true
			if v.Value != "us-east-1" {
				t.Errorf("expected region value to be 'us-east-1', got %q", v.Value)
			}
			if v.IsHCL {
				t.Errorf("expected region IsHCL to be false, got true")
			}
		}
	}
	if !found {
		t.Error("region variable not found")
	}

	// Check instance_count
	found = false
	for _, v := range vars {
		if v.Key == "instance_count" {
			found = true
			if v.Value != "3" {
				t.Errorf("expected instance_count value to be '3', got %q", v.Value)
			}
			if v.IsHCL {
				t.Errorf("expected instance_count IsHCL to be false, got true")
			}
		}
	}
	if !found {
		t.Error("instance_count variable not found")
	}

	// Check enable_monitoring
	found = false
	for _, v := range vars {
		if v.Key == "enable_monitoring" {
			found = true
			if v.Value != "true" {
				t.Errorf("expected enable_monitoring value to be 'true', got %q", v.Value)
			}
			if v.IsHCL {
				t.Errorf("expected enable_monitoring IsHCL to be false, got true")
			}
		}
	}
	if !found {
		t.Error("enable_monitoring variable not found")
	}
}

func TestParseFile_ComplexHCL(t *testing.T) {
	vars, err := ParseFile(filepath.Join("testdata", "complex.tfvars"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(vars) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(vars))
	}

	// Check region (simple type)
	found := false
	for _, v := range vars {
		if v.Key == "region" {
			found = true
			if v.Value != "us-east-1" {
				t.Errorf("expected region value to be 'us-east-1', got %q", v.Value)
			}
			if v.IsHCL {
				t.Errorf("expected region IsHCL to be false, got true")
			}
		}
	}
	if !found {
		t.Error("region variable not found")
	}

	// Check tags (complex type - map)
	found = false
	for _, v := range vars {
		if v.Key == "tags" {
			found = true
			if !v.IsHCL {
				t.Errorf("expected tags IsHCL to be true, got false")
			}
			// Value should be HCL source code
			if v.Value == "" {
				t.Error("expected tags value to be non-empty HCL source")
			}
		}
	}
	if !found {
		t.Error("tags variable not found")
	}

	// Check subnets (complex type - list)
	found = false
	for _, v := range vars {
		if v.Key == "subnets" {
			found = true
			if !v.IsHCL {
				t.Errorf("expected subnets IsHCL to be true, got false")
			}
			// Value should be HCL source code
			if v.Value == "" {
				t.Error("expected subnets value to be non-empty HCL source")
			}
		}
	}
	if !found {
		t.Error("subnets variable not found")
	}
}

func TestParseFile_SimpleJSON(t *testing.T) {
	vars, err := ParseFile(filepath.Join("testdata", "simple.tfvars.json"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(vars) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(vars))
	}

	// Check region
	found := false
	for _, v := range vars {
		if v.Key == "region" {
			found = true
			if v.Value != "us-east-1" {
				t.Errorf("expected region value to be 'us-east-1', got %q", v.Value)
			}
			if v.IsHCL {
				t.Errorf("expected region IsHCL to be false, got true")
			}
		}
	}
	if !found {
		t.Error("region variable not found")
	}

	// Check instance_count
	found = false
	for _, v := range vars {
		if v.Key == "instance_count" {
			found = true
			if v.Value != "3" {
				t.Errorf("expected instance_count value to be '3', got %q", v.Value)
			}
			if v.IsHCL {
				t.Errorf("expected instance_count IsHCL to be false, got true")
			}
		}
	}
	if !found {
		t.Error("instance_count variable not found")
	}

	// Check enable_monitoring
	found = false
	for _, v := range vars {
		if v.Key == "enable_monitoring" {
			found = true
			if v.Value != "true" {
				t.Errorf("expected enable_monitoring value to be 'true', got %q", v.Value)
			}
			if v.IsHCL {
				t.Errorf("expected enable_monitoring IsHCL to be false, got true")
			}
		}
	}
	if !found {
		t.Error("enable_monitoring variable not found")
	}
}

func TestParseFile_ComplexJSON(t *testing.T) {
	vars, err := ParseFile(filepath.Join("testdata", "complex.tfvars.json"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(vars) != 3 {
		t.Fatalf("expected 3 variables, got %d", len(vars))
	}

	// Check region (simple type)
	found := false
	for _, v := range vars {
		if v.Key == "region" {
			found = true
			if v.Value != "us-east-1" {
				t.Errorf("expected region value to be 'us-east-1', got %q", v.Value)
			}
			if v.IsHCL {
				t.Errorf("expected region IsHCL to be false, got true")
			}
		}
	}
	if !found {
		t.Error("region variable not found")
	}

	// Check tags (complex type - map)
	found = false
	for _, v := range vars {
		if v.Key == "tags" {
			found = true
			if !v.IsHCL {
				t.Errorf("expected tags IsHCL to be true, got false")
			}
			// Value should be JSON string
			if v.Value == "" {
				t.Error("expected tags value to be non-empty JSON string")
			}
		}
	}
	if !found {
		t.Error("tags variable not found")
	}

	// Check subnets (complex type - list)
	found = false
	for _, v := range vars {
		if v.Key == "subnets" {
			found = true
			if !v.IsHCL {
				t.Errorf("expected subnets IsHCL to be true, got false")
			}
			// Value should be JSON string
			if v.Value == "" {
				t.Error("expected subnets value to be non-empty JSON string")
			}
		}
	}
	if !found {
		t.Error("subnets variable not found")
	}
}

func TestParseFile_InvalidHCL(t *testing.T) {
	_, err := ParseFile(filepath.Join("testdata", "invalid.tfvars"))
	if err == nil {
		t.Fatal("expected error for invalid HCL, got nil")
	}
}

func TestParseFile_InvalidJSON(t *testing.T) {
	_, err := ParseFile(filepath.Join("testdata", "invalid.json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseFile_UnsupportedExtension(t *testing.T) {
	_, err := ParseFile("test.txt")
	if err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
}

func TestParseFile_FileNotFound(t *testing.T) {
	_, err := ParseFile(filepath.Join("testdata", "nonexistent.tfvars"))
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}
