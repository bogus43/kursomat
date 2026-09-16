package cache

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"kursomat/internal/models"
)

func TestLegacyQueryMappingsRequireRevalidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE query_cache (
currency_code TEXT NOT NULL, requested_date TEXT NOT NULL,
effective_rate_date TEXT NOT NULL, updated_at TEXT NOT NULL,
PRIMARY KEY(currency_code, requested_date));
INSERT INTO query_cache VALUES ('USD','2026-04-14','2026-01-02','2026-04-14T12:00:00Z');`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rate := models.NBPRate{Currency: "USD", EffectiveRateDate: "2026-01-02", Mid: 4}
	if err := s.StoreHistoricalRates("USD", []models.NBPRate{rate}); err != nil {
		t.Fatal(err)
	}
	if _, found, err := s.GetByQuery("USD", "2026-04-14"); err != nil || found {
		t.Fatalf("legacy mapping trusted: found=%v err=%v", found, err)
	}
	info, err := s.Info()
	if err != nil || info.Entries != 1 || info.QueryMappings != 1 {
		t.Fatalf("migration lost data: %+v %v", info, err)
	}
	if err := s.StoreResolvedRate("USD", "2026-04-14", rate); err != nil {
		t.Fatal(err)
	}
	if _, found, err := s.GetByQuery("USD", "2026-04-14"); err != nil || !found {
		t.Fatalf("verified mapping missing: found=%v err=%v", found, err)
	}
}

func TestInvalidBatchCannotPartiallyPersist(t *testing.T) {
	s, err := NewFileStore(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	err = s.StoreHistoricalRates("USD", []models.NBPRate{
		{Currency: "USD", EffectiveRateDate: "2026-04-13", Mid: 4},
		{Currency: "USD", EffectiveRateDate: "2026-04-14", Mid: 0},
	})
	if err == nil {
		t.Fatal("invalid batch accepted")
	}
	info, err := s.Info()
	if err != nil || info.Entries != 0 {
		t.Fatalf("partial batch persisted: %+v %v", info, err)
	}
}

func TestProvisionalMappingCannotBecomePermanent(t *testing.T) {
	s, err := NewFileStore(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	today, err := time.Parse("2006-01-02", models.NBPToday())
	if err != nil {
		t.Fatal(err)
	}
	rate := models.NBPRate{Currency: "USD", EffectiveRateDate: today.AddDate(0, 0, -1).Format("2006-01-02"), Mid: 4}
	if err := s.StoreResolvedRate("USD", models.NBPToday(), rate); err != nil {
		t.Fatal(err)
	}
	// This must remain unverified even after the calendar day changes.
	if _, found, err := s.GetByQuery("USD", models.NBPToday()); found || err != nil {
		t.Fatalf("provisional mapping trusted: %v %v", found, err)
	}
}
