package models

import (
	"os"
	"path/filepath"
)

const (
	DefaultTimeoutSeconds = 10
	DefaultRetryCount     = 2
	DefaultMaxLookback    = 92
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
	cacheDir, err := os.UserCacheDir()
	if err != nil || cacheDir == "" {
		return filepath.Join(".", "cache", "kursownik-nbp-cache.json")
	}
	return filepath.Join(cacheDir, "kursownik-nbp", "cache.json")
}
