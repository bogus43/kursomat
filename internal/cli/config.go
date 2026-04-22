package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kursomat/internal/models"
)

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

	var parsed models.AppConfig
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
	cfg.LastFromDate = parsed.LastFromDate
	cfg.LastConverterDate = parsed.LastConverterDate
	cfg.Normalize()

	if err := ensureDir(filepath.Dir(cfg.CachePath), "katalog danych"); err != nil {
		return cfg, err
	}
	if err := ensureCacheFile(cfg.CachePath); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func SaveConfig(cfg models.AppConfig) error {
	path := models.DefaultConfigPath()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("nie udało się przygotować pliku konfiguracyjnego: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("nie udało się zapisać pliku konfiguracyjnego: %w", err)
	}
	return nil
}

func ensureConfigFile(path string, cfg models.AppConfig) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
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
