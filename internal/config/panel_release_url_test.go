package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigUsesDefaultPanelReleaseURL(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configFile, []byte("port: 8080\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if got := cfg.RemoteManagement.PanelReleaseURL; got != DefaultPanelReleaseURL {
		t.Fatalf("default panel release url = %q, want %q", got, DefaultPanelReleaseURL)
	}
}

func TestLoadConfigReadsPanelReleaseURL(t *testing.T) {
	const cdnURL = "https://a1.dxycdn.com/gitrepo/ai-proxy-management/dist/index.html"
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte("remote-management:\n  panel-release-url: \"" + cdnURL + "\"\n")
	if err := os.WriteFile(configFile, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if got := cfg.RemoteManagement.PanelReleaseURL; got != cdnURL {
		t.Fatalf("panel release url = %q, want %q", got, cdnURL)
	}
}
