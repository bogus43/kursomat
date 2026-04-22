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

func TestSaveConfigAtPathPersistsToCustomLocation(t *testing.T) {
	tmp := t.TempDir()
	customConfig := filepath.Join(tmp, "config", "custom.json")
	customCache := filepath.Join(tmp, "data", "custom.db")

	cfg := models.DefaultConfig()
	cfg.CachePath = customCache
	cfg.TimeoutSeconds = 25
	cfg.RetryCount = 4
	cfg.MaxLookbackDays = 31
	cfg.Verbose = true
	cfg.LastFromDate = "2026-03-01"
	cfg.LastConverterDate = "2026-04-21"

	if err := SaveConfigAtPath(customConfig, cfg); err != nil {
		t.Fatalf("SaveConfigAtPath() error = %v", err)
	}

	runtimeCfg, err := LoadRuntimeConfig(customConfig)
	if err != nil {
		t.Fatalf("LoadRuntimeConfig() error = %v", err)
	}

	if runtimeCfg.Path != customConfig {
		t.Fatalf("expected config path %q, got %q", customConfig, runtimeCfg.Path)
	}
	if runtimeCfg.App.CachePath != customCache {
		t.Fatalf("expected cache path %q, got %q", customCache, runtimeCfg.App.CachePath)
	}
	if runtimeCfg.App.TimeoutSeconds != 25 {
		t.Fatalf("expected timeout 25, got %d", runtimeCfg.App.TimeoutSeconds)
	}
	if runtimeCfg.App.RetryCount != 4 {
		t.Fatalf("expected retry 4, got %d", runtimeCfg.App.RetryCount)
	}
	if runtimeCfg.App.MaxLookbackDays != 31 {
		t.Fatalf("expected lookback 31, got %d", runtimeCfg.App.MaxLookbackDays)
	}
	if !runtimeCfg.App.Verbose {
		t.Fatalf("expected verbose=true")
	}
	if runtimeCfg.App.LastFromDate != "2026-03-01" {
		t.Fatalf("expected last from date to persist, got %q", runtimeCfg.App.LastFromDate)
	}
	if runtimeCfg.App.LastConverterDate != "2026-04-21" {
		t.Fatalf("expected last converter date to persist, got %q", runtimeCfg.App.LastConverterDate)
	}
}
