package nbp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"kursomat/internal/models"
)

func TestVerifiedWeekendCacheIsReused(t *testing.T) {
	calls := 0
	client := auditClient(`{"table":"A","code":"USD","rates":[{"no":"070/A/NBP/2026","effectiveDate":"2026-04-10","mid":3.6015}]}`, &calls)
	service := NewService(client, auditStore(t))
	date := time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		r, err := service.GetRate(context.Background(), "USD", date)
		if err != nil || r.EffectiveRateDate != "2026-04-10" || r.Mid != 3.6015 {
			t.Fatalf("%+v %v", r, err)
		}
	}
	if calls != 1 {
		t.Fatalf("API calls=%d, want 1", calls)
	}
}

func TestTodaysFallbackIsRefreshed(t *testing.T) {
	date, err := time.Parse("2006-01-02", models.NBPToday())
	if err != nil {
		t.Fatal(err)
	}
	s := auditStore(t)
	if err := s.StoreResolvedRate("USD", models.NBPToday(), models.NBPRate{Currency: "USD", EffectiveRateDate: date.AddDate(0, 0, -1).Format("2006-01-02"), Mid: 4}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := auditClient(`{"table":"A","code":"USD","rates":[{"no":"test","effectiveDate":"`+models.NBPToday()+`","mid":3.5}]}`, &calls)
	r, err := NewService(client, s).GetRate(context.Background(), "USD", date)
	if err != nil || r.Mid != 3.5 || calls != 1 {
		t.Fatalf("%+v calls=%d err=%v", r, calls, err)
	}
}

func TestLookbackContinuesAcrossEmptyWindows(t *testing.T) {
	var paths []string
	client := NewClient(ClientConfig{MaxLookbackDays: 100, HTTPClient: &http.Client{Transport: auditTransport(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		status, body := 404, "no data"
		if len(paths) == 2 {
			status, body = 200, `{"table":"A","code":"USD","rates":[{"no":"test","effectiveDate":"2026-01-05","mid":4}]}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}})
	r, err := client.GetRateOnOrBefore(context.Background(), "USD", time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC))
	if err != nil || r.EffectiveRateDate != "2026-01-05" || len(paths) != 2 {
		t.Fatalf("%+v paths=%v err=%v", r, paths, err)
	}
	if !strings.Contains(paths[0], "/2026-01-12/2026-04-14/") || !strings.Contains(paths[1], "/2026-01-04/2026-01-11/") {
		t.Fatalf("gap or overlap: %v", paths)
	}
}

func TestCancelledCacheQueryReturnsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	_, err := NewService(auditClient(`{}`, &calls), auditStore(t)).GetRate(ctx, "USD", time.Now())
	if !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
