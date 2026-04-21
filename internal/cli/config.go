package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kursomat/internal/models"
)

type fileConfig struct {
	CachePath       string `json:"cache_path"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	RetryCount      int    `json:"retry_count"`
	MaxLookbackDays int    `json:"max_lookback_days"`
	Verbose         bool   `json:"verbose"`
}

func LoadConfig(configPath string) (models.AppConfig, error) {
	cfg := models.DefaultConfig()
	cfg.Normalize()

	path := strings.TrimSpace(configPath)
	if path == "" {
		path = models.DefaultConfigPath()
	}

	if err := ensureDir(filepath.Dir(path), "katalog konfiguracji"); err != nil {
		return cfg, err
	}
	if err := ensureDir(filepath.Dir(cfg.CachePath), "katalog danych"); err != nil {
		return cfg, err
	}
	if err := ensureConfigFile(path, cfg); err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("nie udało się odczytać pliku konfiguracyjnego: %w", err)
	}

	var parsed fileConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		return cfg, fmt.Errorf("niepoprawny plik konfiguracyjny: %w", err)
	}

	if parsed.CachePath != "" {
		cfg.CachePath = parsed.CachePath
	}
	if parsed.TimeoutSeconds > 0 {
		cfg.TimeoutSeconds = parsed.TimeoutSeconds
	}
	if parsed.RetryCount >= 0 {
		cfg.RetryCount = parsed.RetryCount
	}
	if parsed.MaxLookbackDays > 0 {
		cfg.MaxLookbackDays = parsed.MaxLookbackDays
	}
	cfg.Verbose = parsed.Verbose
	cfg.Normalize()

	if err := ensureDir(filepath.Dir(cfg.CachePath), "katalog danych"); err != nil {
		return cfg, err
	}
	if err := ensureCacheFile(cfg.CachePath); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func ensureConfigFile(path string, cfg models.AppConfig) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	payload := fileConfig{
		CachePath:       cfg.CachePath,
		TimeoutSeconds:  cfg.TimeoutSeconds,
		RetryCount:      cfg.RetryCount,
		MaxLookbackDays: cfg.MaxLookbackDays,
		Verbose:         cfg.Verbose,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("nie udało się przygotować domyślnego pliku konfiguracyjnego: %w", err)
	}
	data = append(data, '\n')

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("nie udało się utworzyć pliku konfiguracyjnego: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("nie udało się zapisać domyślnego pliku konfiguracyjnego: %w", err)
	}
	return nil
}

func ensureCacheFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("nie udało się utworzyć pliku cache: %w", err)
	}
	defer file.Close()

	return nil
}

func ensureDir(path, label string) error {
	if strings.TrimSpace(path) == "" || path == "." {
		return nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("nie udało się utworzyć %s (%s): %w", label, path, err)
	}
	return nil
}
