package cache

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"kursomat/internal/models"
)

func TestFileStoreRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "cache.db")
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	defer store.Close()

	if err := store.StoreResolvedRate("USD", "2026-04-14", models.NBPRate{
		Currency:          "USD",
		EffectiveRateDate: "2026-04-13",
		Mid:               3.8123,
		TableNo:           "071/A/NBP/2026",
	}); err != nil {
		t.Fatalf("StoreResolvedRate() error = %v", err)
	}

	got, found, err := store.GetByQuery("USD", "2026-04-14")
	if err != nil {
		t.Fatalf("GetByQuery() error = %v", err)
	}
	if !found {
		t.Fatalf("GetByQuery() expected found=true")
	}
	if got.EffectiveRateDate != "2026-04-13" {
		t.Fatalf("unexpected effective date: %s", got.EffectiveRateDate)
	}
	if got.Source != "cache" {
		t.Fatalf("expected source=cache, got %s", got.Source)
	}
}

func TestFileStoreCurrencyCache(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "cache.db")
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	defer store.Close()

	err = store.StoreCurrencies([]models.Currency{
		{Code: "USD", Name: "dolar amerykanski"},
		{Code: "EUR", Name: "euro"},
	})
	if err != nil {
		t.Fatalf("StoreCurrencies() error = %v", err)
	}

	currencies, found, err := store.GetCurrencies()
	if err != nil {
		t.Fatalf("GetCurrencies() error = %v", err)
	}
	if !found || len(currencies) != 2 {
		t.Fatalf("expected 2 cached currencies, got found=%v len=%d", found, len(currencies))
	}
}

func TestNewFileStoreCreatesMissingCacheFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "data", "cache.db")
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected cache file to not exist before test")
	}

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	defer store.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected cache file to be created: %v", err)
	}
}

func TestFileStoreCorruptedCache(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "cache.db")
	if err := os.WriteFile(path, []byte("{invalid-json"), 0o644); err != nil {
		t.Fatalf("cannot write corrupted cache: %v", err)
	}

	_, err := NewFileStore(path)
	if !errors.Is(err, ErrCorruptedCache) {
		t.Fatalf("expected ErrCorruptedCache, got %v", err)
	}
}

func TestFileStoreClear(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "cache.db")
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	defer store.Close()
	if err := store.StoreResolvedRate("EUR", "2026-04-14", models.NBPRate{
		Currency:          "EUR",
		EffectiveRateDate: "2026-04-14",
		Mid:               4.2551,
		TableNo:           "072/A/NBP/2026",
	}); err != nil {
		t.Fatalf("StoreResolvedRate() error = %v", err)
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}
	_, found, err := store.GetByQuery("EUR", "2026-04-14")
	if err != nil {
		t.Fatalf("GetByQuery() error = %v", err)
	}
	if found {
		t.Fatalf("expected found=false after clear")
	}

	currencies, found, err := store.GetCurrencies()
	if err != nil {
		t.Fatalf("GetCurrencies() error = %v", err)
	}
	if found || len(currencies) != 0 {
		t.Fatalf("expected no currencies after clear")
	}
}
