package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	path := stringsTrim(configPath)
	if path == "" {
		path = detectDefaultConfigPath()
	}
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("nie udało się odczytać pliku konfiguracyjnego: %w", err)
	}
	if len(data) == 0 {
		return cfg, nil
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
	return cfg, nil
}

func detectDefaultConfigPath() string {
	candidates := []string{
		"kursownik-nbp.json",
		filepath.Join(".", "config", "kursownik-nbp.json"),
	}
	if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
		candidates = append(candidates, filepath.Join(cfgDir, "kursownik-nbp", "config.json"))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != ' ' && last != '\t' && last != '\n' && last != '\r' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}
