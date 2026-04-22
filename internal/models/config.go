package models

import (
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
	return filepath.Join(".", DefaultDataDir, DefaultCacheFileName)
}

func DefaultConfigPath() string {
	return filepath.Join(".", DefaultConfigDir, DefaultConfigFileName)
}
