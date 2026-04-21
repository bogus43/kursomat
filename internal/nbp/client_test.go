package nbp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPickLatestRate(t *testing.T) {
	t.Parallel()

	payload := rateRangeResponse{
		Code: "USD",
		Rates: []nbpRateRow{
			{No: "070/A/NBP/2026", EffectiveDate: "2026-04-10", Mid: 3.8011},
			{No: "071/A/NBP/2026", EffectiveDate: "2026-04-13", Mid: 3.8123},
		},
	}
	requestedDate, _ := time.Parse("2006-01-02", "2026-04-14")

	got, err := pickLatestRate(payload, "USD", requestedDate)
	if err != nil {
		t.Fatalf("pickLatestRate() error = %v", err)
	}
	if got.EffectiveRateDate != "2026-04-13" {
		t.Fatalf("expected effective date 2026-04-13, got %s", got.EffectiveRateDate)
	}
	if got.TableNo != "071/A/NBP/2026" {
		t.Fatalf("expected table no 071/A/NBP/2026, got %s", got.TableNo)
	}
}

func TestParseRateRangeResponse(t *testing.T) {
	t.Parallel()

	jsonPayload := []byte(`{"table":"A","code":"USD","rates":[{"no":"071/A/NBP/2026","effectiveDate":"2026-04-13","mid":3.8123}]}`)
	parsed, err := parseRateRangeResponse(jsonPayload)
	if err != nil {
		t.Fatalf("parseRateRangeResponse() error = %v", err)
	}
	if parsed.Code != "USD" {
		t.Fatalf("expected code USD, got %s", parsed.Code)
	}
	if len(parsed.Rates) != 1 {
		t.Fatalf("expected 1 rate, got %d", len(parsed.Rates))
	}
	if _, err := parseRateRangeResponse([]byte("{invalid")); err == nil {
		t.Fatalf("expected parsing error for invalid JSON")
	}
}

func TestGetRateOnOrBeforeWithRetry(t *testing.T) {
	t.Parallel()

	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			http.Error(w, "temporary error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"table":"A",
			"code":"USD",
			"rates":[
				{"no":"070/A/NBP/2026","effectiveDate":"2026-04-10","mid":3.8011},
				{"no":"071/A/NBP/2026","effectiveDate":"2026-04-13","mid":3.8123}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL:         server.URL,
		RetryCount:      1,
		MaxLookbackDays: 92,
		HTTPClient:      server.Client(),
	})
	requestedDate, _ := time.Parse("2006-01-02", "2026-04-14")
	got, err := client.GetRateOnOrBefore(context.Background(), "USD", requestedDate)
	if err != nil {
		t.Fatalf("GetRateOnOrBefore() error = %v", err)
	}
	if got.EffectiveRateDate != "2026-04-13" {
		t.Fatalf("expected effective date 2026-04-13, got %s", got.EffectiveRateDate)
	}
	if attempt != 2 {
		t.Fatalf("expected 2 attempts (retry), got %d", attempt)
	}
}

func TestGetRateOnOrBeforeNoData(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Brak danych", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL:         server.URL,
		RetryCount:      1,
		MaxLookbackDays: 92,
		HTTPClient:      server.Client(),
	})
	requestedDate, _ := time.Parse("2006-01-02", "1990-01-01")
	_, err := client.GetRateOnOrBefore(context.Background(), "USD", requestedDate)
	if !errors.Is(err, ErrNoData) {
		t.Fatalf("expected ErrNoData, got %v", err)
	}
}

func TestWaitBackoffCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := waitBackoff(ctx, 2*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("waitBackoff should return quickly when context is cancelled")
	}
}
