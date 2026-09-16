package appcore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"kursomat/internal/nbp"
)

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestConversionThroughNBPAndSQLite(t *testing.T) {
	s := newTestService(t)
	calls := 0
	client := nbp.NewClient(nbp.ClientConfig{HTTPClient: &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if !strings.Contains(r.URL.Path, "/rates/A/USD/") {
			t.Fatalf("unexpected request: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Request: r, Body: io.NopCloser(strings.NewReader(`{"table":"A","code":"USD","rates":[{"no":"071/A/NBP/2026","effectiveDate":"2026-04-14","mid":3.6015}]}`))}, nil
	})}})
	s.backend.provider = nbp.NewService(client, s.backend.store)
	for _, tc := range []struct {
		direction string
		want      float64
	}{
		{"pln_to_foreign", 27.7662}, {"foreign_to_pln", 360.15},
	} {
		// Exercise the same JSON contract as the desktop bridge.
		var request ConvertRequest
		if err := json.Unmarshal([]byte(`{"currency":"usd","amount":100,"date":"2026-04-14","direction":"`+tc.direction+`"}`), &request); err != nil {
			t.Fatal(err)
		}
		r, err := s.Convert(context.Background(), request)
		if err != nil || r.TargetAmount != tc.want || r.Rate.EffectiveRateDate != "2026-04-14" {
			t.Fatalf("result=%+v err=%v", r, err)
		}
		if _, err := json.Marshal(r); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("repeat conversion fetched API %d times", calls)
	}
	history, err := s.CurrencyHistory("USD", -1)
	if err != nil || len(history) != 1 || history[0].Mid != 3.6015 {
		t.Fatalf("history=%+v err=%v", history, err)
	}
}

func TestLiveNBPConversion(t *testing.T) {
	if os.Getenv("KURSOMAT_LIVE_NBP") != "1" {
		t.Skip("set KURSOMAT_LIVE_NBP=1 for the external NBP smoke test")
	}
	s := newTestService(t)
	client := nbp.NewClient(nbp.ClientConfig{Timeout: 20 * time.Second})
	defer client.Close()
	s.backend.provider = nbp.NewService(client, s.backend.store)
	for _, direction := range []string{"pln_to_foreign", "foreign_to_pln"} {
		r, err := s.Convert(context.Background(), ConvertRequest{Currency: "USD", Amount: 100, Date: "2026-04-14", Direction: direction})
		if err != nil {
			t.Fatal(err)
		}
		want := 27.7662
		if direction == "foreign_to_pln" {
			want = 360.15
		}
		if r.TargetAmount != want || r.Rate.Mid != 3.6015 || r.Rate.TableNo != "071/A/NBP/2026" {
			t.Fatalf("unexpected live conversion: %+v", r)
		}
		t.Logf("%s: %g (%s)", direction, r.TargetAmount, r.Rate.Source)
	}
}
