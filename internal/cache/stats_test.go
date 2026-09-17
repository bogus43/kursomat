package cache

import (
	"path/filepath"
	"reflect"
	"testing"

	"kursomat/internal/models"
)

func TestListCurrencyStats(t *testing.T) {
	s, err := NewFileStore(filepath.Join(t.TempDir(), "stats.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if got, err := s.ListCurrencyStats(); err != nil || len(got) != 0 {
		t.Fatalf("empty: %v, %v", got, err)
	}
	if err := s.StoreCurrencies([]models.Currency{{Code: "USD", Name: "dollar"}, {Code: "EUR", Name: "euro"}}); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"USD", "CHF"} {
		if err := s.StoreHistoricalRates(code, []models.NBPRate{
			{Currency: code, EffectiveRateDate: "2024-01-03", Mid: 4},
			{Currency: code, EffectiveRateDate: "2024-01-02", Mid: 3},
		}); err != nil {
			t.Fatal(err)
		}
	}
	want := []CurrencyStat{
		{Code: "CHF", Name: "CHF", RateCount: 2, FirstDate: "2024-01-02", LastDate: "2024-01-03"},
		{Code: "EUR", Name: "euro", RateCount: 0},
		{Code: "USD", Name: "dollar", RateCount: 2, FirstDate: "2024-01-02", LastDate: "2024-01-03"},
	}
	got, err := s.ListCurrencyStats()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("stats = %+v, %v; want %+v", got, err, want)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListCurrencyStats(); err == nil {
		t.Fatal("closed database must return an error")
	}
}
