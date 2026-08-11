package models

import (
	"os"
	"path/filepath"
)

const (
	DefaultTimeoutSeconds = 10
	DefaultRetryCount     = 2
	DefaultMaxLookback    = 92
	DefaultDataDir        = "data"
	DefaultConfigDir      = "config"
	DefaultCacheFileName  = "kursomat.db"
	DefaultConfigFileName = "kursomat.json"
	DefaultAppDirName     = "kursomat"
)

type AppConfig struct {
	CachePath         string `json:"cache_path"`
	TimeoutSeconds    int    `json:"timeout_seconds"`
	RetryCount        int    `json:"retry_count"`
	MaxLookbackDays   int    `json:"max_lookback_days"`
	Verbose           bool   `json:"verbose"`
	LastFromDate      string `json:"last_from_date"`
	LastConverterDate string `json:"last_converter_date"`
}

func DefaultConfig() AppConfig {
	return AppConfig{
		CachePath:       DefaultCachePath(),
		TimeoutSeconds:  DefaultTimeoutSeconds,
		RetryCount:      DefaultRetryCount,
		MaxLookbackDays: DefaultMaxLookback,
		Verbose:         false,
	}
}

func (c *AppConfig) Normalize() {
	if c.CachePath == "" {
		c.CachePath = DefaultCachePath()
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = DefaultTimeoutSeconds
	}
	if c.RetryCount < 0 {
		c.RetryCount = DefaultRetryCount
	}
	if c.MaxLookbackDays <= 0 {
		c.MaxLookbackDays = DefaultMaxLookback
	}
}

func DefaultCachePath() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		return filepath.Join(".", DefaultDataDir, DefaultCacheFileName)
	}
	return filepath.Join(base, DefaultAppDirName, DefaultCacheFileName)
}

func DefaultConfigPath() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return filepath.Join(".", DefaultConfigDir, DefaultConfigFileName)
	}
	return filepath.Join(base, DefaultAppDirName, DefaultConfigFileName)
}
