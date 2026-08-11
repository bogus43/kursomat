package appcore

import (
	"context"
	"math"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"kursomat/internal/cache"
	"kursomat/internal/models"
)

type fakeRateProvider struct {
	rate        models.RateResult
	importCalls int
}

type blockingRateProvider struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (f *blockingRateProvider) GetCurrencies(context.Context) ([]models.Currency, error) {
	return []models.Currency{{Code: "EUR", Name: "euro"}}, nil
}

func (f *blockingRateProvider) GetRate(context.Context, string, time.Time) (models.RateResult, error) {
	return models.RateResult{}, nil
}

func (f *blockingRateProvider) ImportRateRangeChunk(ctx context.Context, _ string, _, _ time.Time) (int, error) {
	f.once.Do(func() { close(f.started) })
	select {
	case <-f.release:
		return 1, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (f *fakeRateProvider) GetCurrencies(context.Context) ([]models.Currency, error) {
	return []models.Currency{{Code: "EUR", Name: "euro"}}, nil
}

func (f *fakeRateProvider) GetRate(context.Context, string, time.Time) (models.RateResult, error) {
	return f.rate, nil
}

func (f *fakeRateProvider) ImportRateRangeChunk(context.Context, string, time.Time, time.Time) (int, error) {
	f.importCalls++
	return 10, nil
}

func TestConvertSupportsBothDirections(t *testing.T) {
	t.Parallel()
	service := newTestService(t)

	plnResult, err := service.Convert(context.Background(), ConvertRequest{
		Currency: "eur", Amount: 100, Date: "2026-04-14", Direction: "pln_to_foreign",
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if plnResult.TargetAmount != 25 || plnResult.TargetCurrency != "EUR" {
		t.Fatalf("unexpected PLN conversion: %#v", plnResult)
	}

	foreignResult, err := service.Convert(context.Background(), ConvertRequest{
		Currency: "EUR", Amount: 25, Date: "2026-04-14", Direction: "foreign_to_pln",
	})
	if err != nil {
		t.Fatalf("Convert() reverse error = %v", err)
	}
	if foreignResult.TargetAmount != 100 || foreignResult.TargetCurrency != "PLN" {
		t.Fatalf("unexpected reverse conversion: %#v", foreignResult)
	}
}

func TestConvertRoundsTargetAmountInBackend(t *testing.T) {
	t.Parallel()
	service := newTestService(t)
	service.backend.provider.(*fakeRateProvider).rate.Mid = 3

	result, err := service.Convert(context.Background(), ConvertRequest{
		Currency: "EUR", Amount: 100, Date: "2026-04-14", Direction: "pln_to_foreign",
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if result.TargetAmount != 33.3333 {
		t.Fatalf("TargetAmount = %v, want 33.3333", result.TargetAmount)
	}
}

func TestConvertRejectsNonFiniteAmount(t *testing.T) {
	t.Parallel()
	service := newTestService(t)

	_, err := service.Convert(context.Background(), ConvertRequest{
		Currency: "EUR", Amount: math.NaN(), Date: "2026-04-14", Direction: "pln_to_foreign",
	})
	if err == nil {
		t.Fatal("Convert() expected an error for NaN amount")
	}
}

func TestConvertRejectsInvalidRate(t *testing.T) {
	t.Parallel()
	service := newTestService(t)
	service.backend.provider.(*fakeRateProvider).rate.Mid = 0

	_, err := service.Convert(context.Background(), ConvertRequest{
		Currency: "EUR", Amount: 100, Date: "2026-04-14", Direction: "pln_to_foreign",
	})
	if err == nil {
		t.Fatal("Convert() expected an error for zero rate")
	}
}

func TestImportRatesChunksLongRangesAndReportsProgress(t *testing.T) {
	t.Parallel()
	service := newTestService(t)
	provider := service.backend.provider.(*fakeRateProvider)
	progressCalls := 0

	summary, err := service.ImportRates(context.Background(), ImportRequest{
		Currencies: []string{"usd", "EUR"},
		StartDate:  "2026-01-01",
		EndDate:    "2026-04-15",
	}, func(progress ImportProgress) {
		progressCalls++
		if progress.CompletedChunks > progress.TotalChunks {
			t.Fatalf("invalid progress: %#v", progress)
		}
	})
	if err != nil {
		t.Fatalf("ImportRates() error = %v", err)
	}
	if summary.ChunkCount != 4 || provider.importCalls != 4 || progressCalls != 4 {
		t.Fatalf("expected four chunks, summary=%#v calls=%d progress=%d", summary, provider.importCalls, progressCalls)
	}
	if summary.RateCount != 40 || summary.CurrencyCount != 2 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestImportRatesRejectsReversedDates(t *testing.T) {
	t.Parallel()
	service := newTestService(t)
	_, err := service.ImportRates(context.Background(), ImportRequest{
		Currencies: []string{"USD"}, StartDate: "2026-04-15", EndDate: "2026-04-14",
	}, nil)
	if err == nil {
		t.Fatal("expected reversed date range error")
	}
}

func TestSettingsSwapDoesNotBlockDashboardDuringImport(t *testing.T) {
	newBackendDir := t.TempDir()
	service := newTestService(t)
	provider := &blockingRateProvider{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	service.backend.provider = provider

	importDone := make(chan error, 1)
	go func() {
		_, err := service.ImportRates(context.Background(), ImportRequest{
			Currencies: []string{"EUR"}, StartDate: "2026-04-14", EndDate: "2026-04-14",
		}, nil)
		importDone <- err
	}()

	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("import did not start")
	}

	var releaseOnce sync.Once
	releaseImport := func() { releaseOnce.Do(func() { close(provider.release) }) }
	defer releaseImport()

	newCachePath := filepath.Join(newBackendDir, "new-cache.db")
	settingsDone := make(chan error, 1)
	go func() {
		_, err := service.UpdateSettings(models.AppConfig{
			CachePath: newCachePath, TimeoutSeconds: 10, RetryCount: 2, MaxLookbackDays: 92,
		})
		settingsDone <- err
	}()

	swapped := make(chan struct{})
	go func() {
		for {
			service.mu.Lock()
			cachePath := service.config.CachePath
			service.mu.Unlock()
			if cachePath == newCachePath {
				close(swapped)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	select {
	case <-swapped:
	case <-time.After(time.Second):
		t.Fatal("settings did not swap to the prepared backend")
	}

	dashboardDone := make(chan struct {
		dashboard Dashboard
		err       error
	}, 1)
	go func() {
		dashboard, err := service.Dashboard()
		dashboardDone <- struct {
			dashboard Dashboard
			err       error
		}{dashboard: dashboard, err: err}
	}()

	select {
	case result := <-dashboardDone:
		if result.err != nil {
			t.Fatalf("Dashboard() error = %v", result.err)
		}
		if result.dashboard.Config.CachePath != newCachePath {
			t.Fatalf("Dashboard() cache path = %q, want %q", result.dashboard.Config.CachePath, newCachePath)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Dashboard() blocked behind the active import")
	}

	releaseImport()
	if err := <-importDone; err != nil {
		t.Fatalf("ImportRates() error = %v", err)
	}
	if err := <-settingsDone; err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	store, err := cache.NewFileStore(filepath.Join(t.TempDir(), "kursomat.db"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	provider := &fakeRateProvider{rate: models.RateResult{
		Currency: "EUR", RequestedDate: "2026-04-14", EffectiveRateDate: "2026-04-14", Mid: 4, Source: "test",
	}}
	configPath := filepath.Join(t.TempDir(), "kursomat.json")
	service := &Service{
		configPath: configPath,
		config: models.AppConfig{
			CachePath: filepath.Join(t.TempDir(), "unused.db"), TimeoutSeconds: 10, RetryCount: 2, MaxLookbackDays: 92,
		},
		backend: &backendState{store: store, provider: provider},
	}
	t.Cleanup(func() { _ = service.Close() })
	return service
}
