package nbp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kursomat/internal/cache"
	"kursomat/internal/models"
)

// Regression tests for the data invariants identified during the audit.
type auditTransport func(*http.Request) (*http.Response, error)

func (f auditTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func auditClient(payload string, calls *int) *Client {
	return NewClient(ClientConfig{HTTPClient: &http.Client{Transport: auditTransport(func(r *http.Request) (*http.Response, error) {
		(*calls)++
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload)), Request: r}, nil
	})}})
}

func auditStore(t *testing.T) cache.Store {
	t.Helper()
	s, err := cache.NewFileStore(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestAuditPartialCacheMustNotHideNewerNBPRate(t *testing.T) {
	s := auditStore(t)
	err := s.StoreHistoricalRates("USD", []models.NBPRate{{Currency: "USD", EffectiveRateDate: "2026-01-02", Mid: 4}})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := auditClient(`{"table":"A","code":"USD","rates":[{"no":"071/A/NBP/2026","effectiveDate":"2026-04-14","mid":3.6015}]}`, &calls)
	result, err := NewService(client, s).GetRate(context.Background(), "USD", time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if result.EffectiveRateDate != "2026-04-14" || result.Mid != 3.6015 {
		t.Fatalf("got date=%s mid=%v API calls=%d; want 2026-04-14 / 3.6015", result.EffectiveRateDate, result.Mid, calls)
	}
}

func TestAuditHistoricalImportMustRefreshResolvedQuery(t *testing.T) {
	s := auditStore(t)
	if err := s.StoreResolvedRate("USD", "2026-04-14", models.NBPRate{Currency: "USD", EffectiveRateDate: "2026-04-10", Mid: 4}); err != nil {
		t.Fatal(err)
	}
	if err := s.StoreHistoricalRates("USD", []models.NBPRate{{Currency: "USD", EffectiveRateDate: "2026-04-14", Mid: 3.6015}}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	result, err := NewService(auditClient(`{}`, &calls), s).GetRate(context.Background(), "USD", time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if result.EffectiveRateDate != "2026-04-14" {
		t.Fatalf("imported a newer rate but resolved query still uses %s", result.EffectiveRateDate)
	}
}

func TestAuditRejectMismatchedResponseCurrency(t *testing.T) {
	for _, mode := range []string{"single", "range"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := auditClient(`{"table":"A","code":"EUR","rates":[{"no":"071/A/NBP/2026","effectiveDate":"2026-04-14","mid":4.25}]}`, &calls)
			date := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
			var err error
			if mode == "single" {
				_, err = client.GetRateOnOrBefore(context.Background(), "USD", date)
			} else {
				_, err = client.GetRatesInRange(context.Background(), "USD", date, date)
			}
			if err == nil {
				t.Fatal("EUR response accepted as USD")
			}
		})
	}
}

func TestAuditImportRejectsInvalidRowsBeforePersistence(t *testing.T) {
	for _, tc := range []struct{ name, date, mid string }{
		{"zero", "2026-04-14", "0"},
		{"negative", "2026-04-14", "-1"},
		{"null", "2026-04-14", "null"},
		{"bad_date", "2026-02-30", "4"},
		{"outside_range", "2026-04-15", "4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := auditStore(t)
			calls := 0
			payload := fmt.Sprintf(`{"table":"A","code":"USD","rates":[{"no":"071/A/NBP/2026","effectiveDate":%q,"mid":%s}]}`, tc.date, tc.mid)
			date := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
			count, err := NewService(auditClient(payload, &calls), s).ImportRateRangeChunk(context.Background(), "USD", date, date)
			if err == nil {
				t.Errorf("invalid row accepted, imported count=%d", count)
			}
			info, infoErr := s.Info()
			if infoErr != nil {
				t.Fatal(infoErr)
			}
			if info.Entries != 0 {
				t.Errorf("invalid data persisted: %d rate(s)", info.Entries)
			}
		})
	}
}

func TestAuditLatestRateDoesNotDependOnResponseOrder(t *testing.T) {
	payload := rateRangeResponse{Code: "USD", Rates: []nbpRateRow{
		{EffectiveDate: "2026-04-14", Mid: 3.6015},
		{EffectiveDate: "2026-04-10", Mid: 4},
	}}
	r, err := pickLatestRate(payload, "USD", time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.EffectiveRateDate != "2026-04-14" {
		t.Fatalf("latest returned: %s", r.EffectiveRateDate)
	}
}

func TestAuditLookbackRequestFitsNBPWindow(t *testing.T) {
	client := NewClient(ClientConfig{MaxLookbackDays: 3660, HTTPClient: &http.Client{Transport: auditTransport(func(r *http.Request) (*http.Response, error) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		start, err := time.Parse("2006-01-02", parts[len(parts)-2])
		if err != nil {
			t.Fatal(err)
		}
		end, err := time.Parse("2006-01-02", parts[len(parts)-1])
		if err != nil {
			t.Fatal(err)
		}
		if days := int(end.Sub(start).Hours()/24) + 1; days > 93 {
			t.Errorf("single API request spans %d days; NBP limit is 93", days)
		}
		return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("no data")), Request: r}, nil
	})}})
	_, _ = client.GetRateOnOrBefore(context.Background(), "USD", time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC))
}
