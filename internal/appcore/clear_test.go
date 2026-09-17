package appcore

import (
	"testing"

	"kursomat/internal/models"
)

func TestClearCache(t *testing.T) {
	s := newTestService(t)
	store := s.backend.store
	if err := store.StoreResolvedRate("EUR", "2024-01-03", models.NBPRate{Currency: "EUR", EffectiveRateDate: "2024-01-02", Mid: 4}); err != nil {
		t.Fatal(err)
	}
	if err := store.StoreCurrencies([]models.Currency{{Code: "EUR", Name: "euro"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearCache(); err != nil {
		t.Fatal(err)
	}
	info, err := store.Info()
	if err != nil || info.Entries != 0 || info.QueryMappings != 0 || info.CurrencyCount != 0 {
		t.Fatalf("cache after clear: %+v, %v", info, err)
	}
	if err := s.ClearCache(); err != nil {
		t.Fatalf("clear empty cache: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearCache(); err == nil {
		t.Fatal("closed store must return an error")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearCache(); err == nil {
		t.Fatal("closed service must return an error")
	}
}
