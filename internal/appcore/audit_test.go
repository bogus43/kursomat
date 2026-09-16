package appcore

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestAuditConversionResultRemainsFinite(t *testing.T) {
	for _, tc := range []struct {
		name, direction string
		amount, mid     float64
	}{
		{"multiplication_overflow", "foreign_to_pln", 1e308, 4},
		{"rounding_overflow", "pln_to_foreign", 1e308, 4},
		{"division_overflow", "pln_to_foreign", 1e308, 0.01},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(t)
			s.backend.provider.(*fakeRateProvider).rate.Mid = tc.mid
			r, err := s.Convert(context.Background(), ConvertRequest{Currency: "EUR", Amount: tc.amount, Date: "2026-04-14", Direction: tc.direction})
			if err != nil {
				return
			} // Rejection is a valid outcome for an unrepresentable amount.
			if math.IsNaN(r.TargetAmount) || math.IsInf(r.TargetAmount, 0) {
				t.Errorf("successful conversion returned %v", r.TargetAmount)
			}
			if _, err := json.Marshal(r); err != nil {
				t.Errorf("result cannot cross the JSON bridge: %v", err)
			}
		})
	}
}

func TestAuditInputValidation(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		amount                    float64
		date, currency, direction string
	}{
		{"negative", -1, "2026-04-14", "EUR", "pln_to_foreign"},
		{"nan", math.NaN(), "2026-04-14", "EUR", "pln_to_foreign"},
		{"infinity", math.Inf(1), "2026-04-14", "EUR", "pln_to_foreign"},
		{"invalid_date", 1, "2026-02-30", "EUR", "pln_to_foreign"},
		{"noncanonical_date", 1, "2026-04-4", "EUR", "pln_to_foreign"},
		{"before_archive", 1, "2001-01-01", "EUR", "pln_to_foreign"},
		{"future_date", 1, "9999-12-31", "EUR", "pln_to_foreign"},
		{"invalid_currency", 1, "2026-04-14", "EURO", "pln_to_foreign"},
		{"invalid_direction", 1, "2026-04-14", "EUR", "invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newTestService(t).Convert(context.Background(), ConvertRequest{Currency: tc.currency, Amount: tc.amount, Date: tc.date, Direction: tc.direction})
			if err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

func TestAuditImportChunksAreContiguousAndWithinNBPLimit(t *testing.T) {
	start := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	chunks := buildImportChunks([]string{"USD"}, start, end)
	next := start
	for _, chunk := range chunks {
		if !chunk.start.Equal(next) {
			t.Fatalf("gap/overlap: expected %s got %s", next, chunk.start)
		}
		if days := int(chunk.end.Sub(chunk.start).Hours()/24) + 1; days < 1 || days > 93 {
			t.Fatalf("invalid chunk size: %d", days)
		}
		next = chunk.end.AddDate(0, 0, 1)
	}
	if !next.Equal(end.AddDate(0, 0, 1)) {
		t.Fatal("range not fully covered")
	}
}

func TestAuditKnownConversions(t *testing.T) {
	for _, tc := range []struct {
		name, direction   string
		amount, mid, want float64
	}{
		{"nbp_to_pln", "foreign_to_pln", 100, 3.6015, 360.15},
		{"nbp_from_pln", "pln_to_foreign", 100, 3.6015, 27.7662},
		{"zero", "foreign_to_pln", 0, 3.6015, 0},
		{"below_precision", "foreign_to_pln", 0.00001, 3.6015, 0},
		{"rounding_half_up", "foreign_to_pln", 0.00005, 1, 0.0001},
		{"decimal_tie", "foreign_to_pln", 1.00105, 1, 1.0011},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(t)
			provider := s.backend.provider.(*fakeRateProvider)
			provider.rate.Mid = tc.mid
			provider.rate.Currency = "USD"
			r, err := s.Convert(context.Background(), ConvertRequest{Currency: "USD", Amount: tc.amount, Date: "2026-04-14", Direction: tc.direction})
			if err != nil {
				t.Fatal(err)
			}
			if r.TargetAmount != tc.want {
				t.Fatalf("amount=%v mid=%v: got %v want %v", tc.amount, tc.mid, r.TargetAmount, tc.want)
			}
		})
	}
}
