package appcore

import (
	"os"
	"path/filepath"
	"testing"

	"kursomat/internal/models"
)

func TestLoadRuntimeConfigCreatesExplicitConfigAndDataDirectory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	configPath := filepath.Join(root, "config", "custom.json")
	cachePath := filepath.Join(root, "database", "custom.db")

	cfg := models.DefaultConfig()
	cfg.CachePath = cachePath
	if err := SaveConfigAtPath(configPath, cfg); err != nil {
		t.Fatalf("SaveConfigAtPath() error = %v", err)
	}

	runtimeConfig, err := LoadRuntimeConfig(configPath)
	if err != nil {
		t.Fatalf("LoadRuntimeConfig() error = %v", err)
	}
	if runtimeConfig.Path != configPath || runtimeConfig.App.CachePath != cachePath {
		t.Fatalf("unexpected runtime config: %#v", runtimeConfig)
	}
	if _, err := os.Stat(filepath.Dir(cachePath)); err != nil {
		t.Fatalf("expected cache directory: %v", err)
	}
}

func TestSaveConfigAtPathPersistsDesktopSettings(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "settings", "kursomat.json")
	cfg := models.AppConfig{
		CachePath:         filepath.Join(t.TempDir(), "kursomat.db"),
		TimeoutSeconds:    25,
		RetryCount:        4,
		MaxLookbackDays:   31,
		Verbose:           true,
		LastFromDate:      "2026-03-01",
		LastConverterDate: "2026-04-21",
	}
	if err := SaveConfigAtPath(path, cfg); err != nil {
		t.Fatalf("SaveConfigAtPath() error = %v", err)
	}
	loaded, err := LoadRuntimeConfig(path)
	if err != nil {
		t.Fatalf("LoadRuntimeConfig() error = %v", err)
	}
	if loaded.App != cfg {
		t.Fatalf("settings changed after round trip:\nwant %#v\n got %#v", cfg, loaded.App)
	}
}

func TestNormalizeCurrenciesDeduplicatesAndNormalizesCodes(t *testing.T) {
	t.Parallel()
	codes, err := NormalizeCurrencies([]string{"usd", " EUR ", "USD"})
	if err != nil {
		t.Fatalf("NormalizeCurrencies() error = %v", err)
	}
	if len(codes) != 2 || codes[0] != "USD" || codes[1] != "EUR" {
		t.Fatalf("unexpected codes: %#v", codes)
	}
	if _, err := NormalizeCurrencies([]string{"EU"}); err == nil {
		t.Fatal("expected invalid currency error")
	}
}

func TestLoadRuntimeConfigMakesLegacyCachePathAbsolute(t *testing.T) {
	root := t.TempDir()
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previousDirectory) })

	configPath := filepath.Join(root, "config", "legacy.json")
	cfg := models.DefaultConfig()
	cfg.CachePath = filepath.Join("data", "kursomat.db")
	if err := SaveConfigAtPath(configPath, cfg); err != nil {
		t.Fatalf("SaveConfigAtPath() error = %v", err)
	}
	loaded, err := LoadRuntimeConfig(configPath)
	if err != nil {
		t.Fatalf("LoadRuntimeConfig() error = %v", err)
	}
	expected := filepath.Join(root, "data", "kursomat.db")
	if loaded.App.CachePath != expected {
		t.Fatalf("expected absolute cache path %q, got %q", expected, loaded.App.CachePath)
	}
}
