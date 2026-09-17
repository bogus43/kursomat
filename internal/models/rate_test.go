package models

import (
	"math"
	"testing"
)

func TestNBPRateValidate(t *testing.T) {
	valid := NBPRate{Currency: "EUR", EffectiveRateDate: "2024-02-29", Mid: 4.25}
	for _, tc := range []struct {
		name, currency, date string
		mid                  float64
		wantErr              bool
	}{
		{"valid leap day", "EUR", "2024-02-29", 4.25, false},
		{"currency mismatch", "USD", "2024-02-29", 4.25, true},
		{"lowercase", "eur", "2024-02-29", 4.25, true},
		{"empty currency", "", "2024-02-29", 4.25, true},
		{"invalid leap day", "EUR", "2023-02-29", 4.25, true},
		{"noncanonical date", "EUR", "2024-2-29", 4.25, true},
		{"zero", "EUR", "2024-02-29", 0, true},
		{"negative", "EUR", "2024-02-29", -1, true},
		{"nan", "EUR", "2024-02-29", math.NaN(), true},
		{"positive infinity", "EUR", "2024-02-29", math.Inf(1), true},
		{"negative infinity", "EUR", "2024-02-29", math.Inf(-1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rate := valid
			rate.EffectiveRateDate, rate.Mid = tc.date, tc.mid
			if err := rate.Validate(tc.currency); (err != nil) != tc.wantErr {
				t.Fatalf("Validate() = %v, want error %v", err, tc.wantErr)
			}
		})
	}
}
