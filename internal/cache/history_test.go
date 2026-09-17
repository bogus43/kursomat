package cache

import (
	"path/filepath"
	"reflect"
	"testing"

	"kursomat/internal/models"
)

func TestListCurrencyHistory(t *testing.T) {
	s, err := NewFileStore(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.StoreHistoricalRates("USD", []models.NBPRate{
		{Currency: "USD", EffectiveRateDate: "2024-01-02", Mid: 3, TableNo: "001/A"},
		{Currency: "USD", EffectiveRateDate: "2024-01-04", Mid: 5, TableNo: "003/A"},
		{Currency: "USD", EffectiveRateDate: "2024-01-03", Mid: 4},
	}); err != nil {
		t.Fatal(err)
	}
	want := []CurrencyHistoryEntry{{EffectiveRateDate: "2024-01-04", Mid: 5, TableNo: "003/A"}, {EffectiveRateDate: "2024-01-03", Mid: 4}, {EffectiveRateDate: "2024-01-02", Mid: 3, TableNo: "001/A"}}
	for _, limit := range []int{0, -1, 1, 2, 10} {
		expected := want
		if limit > 0 && limit < len(want) {
			expected = want[:limit]
		}
		got, err := s.ListCurrencyHistory("usd", limit)
		if err != nil || !reflect.DeepEqual(got, expected) {
			t.Fatalf("limit %d: got %+v, %v; want %+v", limit, got, err, expected)
		}
	}
	for _, code := range []string{"EUR", "USD' OR 1=1 --"} {
		if got, err := s.ListCurrencyHistory(code, 0); err != nil || len(got) != 0 {
			t.Fatalf("currency %q: %+v, %v", code, got, err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListCurrencyHistory("USD", 1); err == nil {
		t.Fatal("closed database must return an error")
	}
}
