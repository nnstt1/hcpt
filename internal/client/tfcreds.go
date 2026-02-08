package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// findTerraformToken searches for a Terraform CLI token for the given hostname.
// It checks in order: TF_TOKEN_<hostname> env var, credentials.tfrc.json, .terraformrc.
func findTerraformToken(hostname string) string {
	if token := findTokenFromEnv(hostname); token != "" {
		return token
	}
	if token := findTokenFromCredentialsJSON(hostname); token != "" {
		return token
	}
	if token := findTokenFromTerraformRC(hostname); token != "" {
		return token
	}
	return ""
}

// findTokenFromEnv checks for TF_TOKEN_<hostname> environment variable.
// Dots and dashes in hostname are replaced with underscores.
func findTokenFromEnv(hostname string) string {
	envKey := "TF_TOKEN_" + strings.NewReplacer(".", "_", "-", "_").Replace(hostname)
	return os.Getenv(envKey)
}

// credentialsJSON represents the structure of credentials.tfrc.json.
type credentialsJSON struct {
	Credentials map[string]credentialEntry `json:"credentials"`
}

type credentialEntry struct {
	Token string `json:"token"`
}

// findTokenFromCredentialsJSON reads token from ~/.terraform.d/credentials.tfrc.json.
func findTokenFromCredentialsJSON(hostname string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	path := filepath.Join(home, ".terraform.d", "credentials.tfrc.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var creds credentialsJSON
	if err := json.Unmarshal(data, &creds); err != nil {
		return ""
	}

	if entry, ok := creds.Credentials[hostname]; ok {
		return entry.Token
	}
	return ""
}

// terraformRCConfig represents the HCL structure of .terraformrc.
type terraformRCConfig struct {
	Credentials []terraformRCCredential `hcl:"credentials,block"`
	Remain      hcl.Body                `hcl:",remain"`
}

type terraformRCCredential struct {
	Name  string `hcl:"name,label"`
	Token string `hcl:"token"`
}

// findTokenFromTerraformRC reads token from .terraformrc (HCL format).
func findTokenFromTerraformRC(hostname string) string {
	path := terraformRCPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	file, diags := hclsyntax.ParseConfig(data, path, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return ""
	}

	var config terraformRCConfig
	diags = gohcl.DecodeBody(file.Body, nil, &config)
	if diags.HasErrors() {
		return ""
	}

	for _, cred := range config.Credentials {
		if cred.Name == hostname {
			return cred.Token
		}
	}
	return ""
}

// terraformRCPath returns the path to .terraformrc.
// If TF_CLI_CONFIG_FILE is set, it is used instead of the default.
func terraformRCPath() string {
	if p := os.Getenv("TF_CLI_CONFIG_FILE"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".terraformrc")
}
