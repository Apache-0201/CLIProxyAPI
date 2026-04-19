package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigOptionalExtractsAPIKeyAuthIdentityBinding(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	configYAML := []byte(`
api-keys:
  - api-key: sk-client
    name: client
    auth_index: old-runtime-index
    auth_identity: codex:chatgpt:acct-stable
`)
	if err := os.WriteFile(configPath, configYAML, 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfigOptional(configPath, false)
	if err != nil {
		t.Fatalf("LoadConfigOptional() error = %v", err)
	}

	if got := cfg.APIKeyAuthIdentityBindings["sk-client"]; got != "codex:chatgpt:acct-stable" {
		t.Fatalf("auth_identity binding = %q, want %q", got, "codex:chatgpt:acct-stable")
	}
	if got := cfg.APIKeyAuthBindings["sk-client"]; got != "old-runtime-index" {
		t.Fatalf("legacy auth_index binding = %q, want %q", got, "old-runtime-index")
	}
}
