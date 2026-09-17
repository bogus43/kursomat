package nbp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPrefetchRates(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 4, 2, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name                      string
		status, failAt, wantRates int
		wantErr                   bool
	}{
		{"rates", 200, 0, 4, false},
		{"no data", 404, 0, 0, false},
		{"second chunk fails", 500, 2, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			client := NewClient(ClientConfig{HTTPClient: &http.Client{Transport: auditTransport(func(r *http.Request) (*http.Response, error) {
				paths = append(paths, r.URL.Path)
				parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
				code, date := strings.ToUpper(parts[len(parts)-3]), parts[len(parts)-2]
				status := tc.status
				if tc.failAt > 0 && len(paths) != tc.failAt {
					status = 200
				}
				body := fmt.Sprintf(`{"table":"A","code":%q,"rates":[{"effectiveDate":%q,"mid":4}]}`, code, date)
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})}})
			got, err := NewService(client, nil).PrefetchRates(context.Background(), []string{"USD", "EUR"}, start, end)
			if (err != nil) != tc.wantErr {
				t.Fatalf("summary=%+v err=%v", got, err)
			}
			wantPaths := []string{
				"/api/exchangerates/rates/A/USD/2024-01-01/2024-03-31/",
				"/api/exchangerates/rates/A/USD/2024-04-01/2024-04-02/",
				"/api/exchangerates/rates/A/EUR/2024-01-01/2024-03-31/",
				"/api/exchangerates/rates/A/EUR/2024-04-01/2024-04-02/",
			}
			if tc.wantErr {
				wantPaths = wantPaths[:tc.failAt]
				if got != (PrefetchSummary{}) {
					t.Fatalf("failure returned successful summary: %+v", got)
				}
			} else if got.RateCount != tc.wantRates || got.CurrencyCount != 2 || got.StartDate != "2024-01-01" || got.EndDate != "2024-04-02" {
				t.Fatalf("summary = %+v", got)
			}
			if !reflect.DeepEqual(paths, wantPaths) {
				t.Fatalf("paths = %v; want %v", paths, wantPaths)
			}
		})
	}
	s := NewService(nil, nil)
	if _, err := s.PrefetchRates(context.Background(), []string{"USD"}, end, start); err == nil {
		t.Fatal("reversed range accepted")
	}
	if _, err := s.PrefetchRates(context.Background(), nil, start, end); err == nil {
		t.Fatal("empty currencies accepted")
	}
}
