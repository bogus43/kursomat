package cli

import (
	"os"
	"path/filepath"
	"testing"

	"kursomat/internal/models"
)

func TestLoadConfigCreatesDefaultFoldersAndFiles(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}
	defer func() { _ = os.Chdir(previousWD) }()

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.CachePath != models.DefaultCachePath() {
		t.Fatalf("expected cache path %q, got %q", models.DefaultCachePath(), cfg.CachePath)
	}

	defaultConfigPath := models.DefaultConfigPath()
	if _, err := os.Stat(defaultConfigPath); err != nil {
		t.Fatalf("expected config file to be created at %q: %v", defaultConfigPath, err)
	}
	if _, err := os.Stat(cfg.CachePath); err != nil {
		t.Fatalf("expected cache file to be created at %q: %v", cfg.CachePath, err)
	}
}

func TestLoadConfigCreatesMissingCustomConfigAndCache(t *testing.T) {
	tmp := t.TempDir()
	customConfig := filepath.Join(tmp, "config", "custom.json")

	cfg, err := LoadConfig(customConfig)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if _, err := os.Stat(customConfig); err != nil {
		t.Fatalf("expected custom config file to be created at %q: %v", customConfig, err)
	}
	if _, err := os.Stat(cfg.CachePath); err != nil {
		t.Fatalf("expected cache file to be created at %q: %v", cfg.CachePath, err)
	}
}
