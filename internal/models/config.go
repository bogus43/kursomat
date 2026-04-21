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
	DefaultCacheFileName  = "kursownik.db"
	DefaultConfigFileName = "kursownik-nbp.json"
)

type AppConfig struct {
	CachePath       string `json:"cache_path"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	RetryCount      int    `json:"retry_count"`
	MaxLookbackDays int    `json:"max_lookback_days"`
	Verbose         bool   `json:"verbose"`
}

func DefaultConfig() AppConfig {
	return AppConfig{
		CachePath:       defaultCachePath(),
		TimeoutSeconds:  DefaultTimeoutSeconds,
		RetryCount:      DefaultRetryCount,
		MaxLookbackDays: DefaultMaxLookback,
		Verbose:         false,
	}
}

func (c *AppConfig) Normalize() {
	if c.CachePath == "" {
		c.CachePath = defaultCachePath()
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

func defaultCachePath() string {
	return DefaultCachePath()
}

func DefaultCachePath() string {
	return filepath.Join(".", DefaultDataDir, DefaultCacheFileName)
}

func DefaultConfigPath() string {
	return filepath.Join(".", DefaultConfigDir, DefaultConfigFileName)
}
