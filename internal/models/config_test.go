package models

import "testing"

func TestNormalize(t *testing.T) {
	for _, value := range []int{-1, 0} {
		cfg := AppConfig{TimeoutSeconds: value, RetryCount: value, MaxLookbackDays: value}
		cfg.Normalize()
		wantRetries := value
		if value < 0 {
			wantRetries = DefaultRetryCount
		}
		if cfg.CachePath == "" || cfg.TimeoutSeconds != DefaultTimeoutSeconds || cfg.RetryCount != wantRetries || cfg.MaxLookbackDays != DefaultMaxLookback {
			t.Fatalf("Normalize(%d) = %+v", value, cfg)
		}
	}
	want := AppConfig{CachePath: "custom.db", TimeoutSeconds: 30, RetryCount: 0, MaxLookbackDays: 180, Verbose: true, LastFromDate: "2024-01-01", LastConverterDate: "2024-02-01"}
	got := want
	got.Normalize()
	got.Normalize()
	if got != want {
		t.Fatalf("custom settings changed: got %+v, want %+v", got, want)
	}
}
