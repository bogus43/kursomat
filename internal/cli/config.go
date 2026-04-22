package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"kursomat/internal/models"
)

type RuntimeConfig struct {
	Path string
	App  models.AppConfig
}

func LoadConfig(configPath string) (models.AppConfig, error) {
	runtimeCfg, err := LoadRuntimeConfig(configPath)
	if err != nil {
		return models.AppConfig{}, err
	}
	return runtimeCfg.App, nil
}

func LoadRuntimeConfig(configPath string) (RuntimeConfig, error) {
	cfg := models.DefaultConfig()
	cfg.Normalize()

	path := strings.TrimSpace(configPath)
	if path == "" {
		path = models.DefaultConfigPath()
	}

	if err := ensureDir(filepath.Dir(path), "katalog konfiguracji"); err != nil {
		return RuntimeConfig{}, err
	}
	if err := ensureDir(filepath.Dir(cfg.CachePath), "katalog danych"); err != nil {
		return RuntimeConfig{}, err
	}
	if err := ensureConfigFile(path, cfg); err != nil {
		return RuntimeConfig{}, err
	}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")
	v.SetEnvPrefix("KURSOMAT")
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return RuntimeConfig{}, fmt.Errorf("nie udało się odczytać pliku konfiguracyjnego: %w", err)
	}

	if v.IsSet("cache_path") {
		cfg.CachePath = v.GetString("cache_path")
	}
	if v.IsSet("timeout_seconds") {
		cfg.TimeoutSeconds = v.GetInt("timeout_seconds")
	}
	if v.IsSet("retry_count") {
		cfg.RetryCount = v.GetInt("retry_count")
	}
	if v.IsSet("max_lookback_days") {
		cfg.MaxLookbackDays = v.GetInt("max_lookback_days")
	}
	if v.IsSet("verbose") {
		cfg.Verbose = v.GetBool("verbose")
	}
	if v.IsSet("last_from_date") {
		cfg.LastFromDate = v.GetString("last_from_date")
	}
	if v.IsSet("last_converter_date") {
		cfg.LastConverterDate = v.GetString("last_converter_date")
	}
	cfg.Normalize()

	if err := ensureDir(filepath.Dir(cfg.CachePath), "katalog danych"); err != nil {
		return RuntimeConfig{}, err
	}
	if err := ensureCacheFile(cfg.CachePath); err != nil {
		return RuntimeConfig{}, err
	}

	return RuntimeConfig{
		Path: path,
		App:  cfg,
	}, nil
}

func SaveConfig(cfg models.AppConfig) error {
	return SaveConfigAtPath(models.DefaultConfigPath(), cfg)
}

func SaveConfigAtPath(path string, cfg models.AppConfig) error {
	cfg.Normalize()
	path = strings.TrimSpace(path)
	if path == "" {
		path = models.DefaultConfigPath()
	}
	if err := ensureDir(filepath.Dir(path), "katalog konfiguracji"); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("nie udało się przygotować pliku konfiguracyjnego: %w", err)
	}
	data = append(data, '\n')

	tmpFile, err := os.CreateTemp(filepath.Dir(path), "kursomat-config-*.tmp")
	if err != nil {
		return fmt.Errorf("nie udało się utworzyć pliku tymczasowego konfiguracji: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("nie udało się zapisać pliku tymczasowego konfiguracji: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("nie udało się zamknąć pliku tymczasowego konfiguracji: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("nie udało się przygotować podmiany pliku konfiguracyjnego: %w", removeErr)
		}
		if renameErr := os.Rename(tmpPath, path); renameErr != nil {
			return fmt.Errorf("nie udało się zapisać pliku konfiguracyjnego: %w", renameErr)
		}
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
