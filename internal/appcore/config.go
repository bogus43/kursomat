package appcore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"kursomat/internal/models"
)

type RuntimeConfig struct {
	Path string
	App  models.AppConfig
}

func LoadRuntimeConfig(configPath string) (RuntimeConfig, error) {
	path := strings.TrimSpace(configPath)
	useDefaultPath := path == ""
	if useDefaultPath {
		path = models.DefaultConfigPath()
	}

	cfg := models.DefaultConfig()
	if useDefaultPath {
		legacyPath := findLegacyConfig()
		if _, err := os.Stat(path); os.IsNotExist(err) && legacyPath != "" {
			legacy, readErr := readConfig(legacyPath, cfg)
			if readErr != nil {
				return RuntimeConfig{}, readErr
			}
			cfg = legacy
		}
	}

	if _, err := os.Stat(path); err == nil {
		loaded, readErr := readConfig(path, cfg)
		if readErr != nil {
			return RuntimeConfig{}, readErr
		}
		cfg = loaded
	} else if !os.IsNotExist(err) {
		return RuntimeConfig{}, fmt.Errorf("nie udało się sprawdzić pliku konfiguracyjnego: %w", err)
	}

	applyEnvironment(&cfg)
	cfg.Normalize()
	if !filepath.IsAbs(cfg.CachePath) {
		absolutePath, err := filepath.Abs(cfg.CachePath)
		if err != nil {
			return RuntimeConfig{}, fmt.Errorf("nie udało się ustalić pełnej ścieżki bazy: %w", err)
		}
		cfg.CachePath = absolutePath
	}
	if err := SaveConfigAtPath(path, cfg); err != nil {
		return RuntimeConfig{}, err
	}
	if err := ensureDir(filepath.Dir(cfg.CachePath), "katalog danych"); err != nil {
		return RuntimeConfig{}, err
	}

	return RuntimeConfig{Path: path, App: cfg}, nil
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
	if err := replaceFile(tmpPath, path); err != nil {
		return err
	}
	return nil
}

func readConfig(path string, defaults models.AppConfig) (models.AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return models.AppConfig{}, fmt.Errorf("nie udało się odczytać pliku konfiguracyjnego: %w", err)
	}
	cfg := defaults
	if err := json.Unmarshal(data, &cfg); err != nil {
		return models.AppConfig{}, fmt.Errorf("nie udało się odczytać ustawień z pliku konfiguracyjnego: %w", err)
	}
	return cfg, nil
}

func findLegacyConfig() string {
	for _, candidate := range []string{
		filepath.Join("config", "kursomat.json"),
		filepath.Join("config", "kursownik-nbp.json"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func applyEnvironment(cfg *models.AppConfig) {
	if value := strings.TrimSpace(os.Getenv("KURSOMAT_CACHE_PATH")); value != "" {
		cfg.CachePath = value
	}
	applyIntEnv("KURSOMAT_TIMEOUT_SECONDS", &cfg.TimeoutSeconds)
	applyIntEnv("KURSOMAT_RETRY_COUNT", &cfg.RetryCount)
	applyIntEnv("KURSOMAT_MAX_LOOKBACK_DAYS", &cfg.MaxLookbackDays)
	if value := strings.TrimSpace(os.Getenv("KURSOMAT_VERBOSE")); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Verbose = parsed
		}
	}
}

func applyIntEnv(name string, target *int) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		*target = parsed
	}
}

func replaceFile(source, target string) error {
	if err := os.Rename(source, target); err == nil {
		return nil
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("nie udało się przygotować podmiany pliku konfiguracyjnego: %w", err)
	}
	if err := os.Rename(source, target); err != nil {
		return fmt.Errorf("nie udało się zapisać pliku konfiguracyjnego: %w", err)
	}
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
