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

	path := filepath.Join(t.TempDir(), "cache.json")
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

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

func TestFileStoreCorruptedCache(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "cache.json")
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

	path := filepath.Join(t.TempDir(), "cache.json")
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
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
}
