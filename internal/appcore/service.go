package appcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kursomat/internal/cache"
	"kursomat/internal/models"
	"kursomat/internal/nbp"
)

const importChunkDays = 90
const currencyPrecision = 10000

type rateProvider interface {
	GetCurrencies(context.Context) ([]models.Currency, error)
	GetRate(context.Context, string, time.Time) (models.RateResult, error)
	ImportRateRangeChunk(context.Context, string, time.Time, time.Time) (int, error)
}

type backendState struct {
	store    cache.Store
	provider rateProvider
	client   io.Closer
	active   sync.WaitGroup
}

type Service struct {
	mu            sync.Mutex
	reconfigureMu sync.Mutex
	configPath    string
	config        models.AppConfig
	backend       *backendState
}

type Dashboard struct {
	ConfigPath string               `json:"config_path"`
	Config     models.AppConfig     `json:"config"`
	Cache      cache.Info           `json:"cache"`
	Currencies []cache.CurrencyStat `json:"currencies"`
}

type ConvertRequest struct {
	Currency  string  `json:"currency"`
	Amount    float64 `json:"amount"`
	Date      string  `json:"date"`
	Direction string  `json:"direction"`
}

type ConversionResult struct {
	SourceAmount   float64           `json:"source_amount"`
	SourceCurrency string            `json:"source_currency"`
	TargetAmount   float64           `json:"target_amount"`
	TargetCurrency string            `json:"target_currency"`
	Rate           models.RateResult `json:"rate"`
}

type ImportRequest struct {
	Currencies []string `json:"currencies"`
	StartDate  string   `json:"start_date"`
	EndDate    string   `json:"end_date"`
}

type ImportProgress struct {
	CompletedChunks int    `json:"completed_chunks"`
	TotalChunks     int    `json:"total_chunks"`
	Currency        string `json:"currency"`
	ChunkStart      string `json:"chunk_start"`
	ChunkEnd        string `json:"chunk_end"`
	RateCount       int    `json:"rate_count"`
}

type ImportSummary struct {
	CurrencyCount int    `json:"currency_count"`
	RateCount     int    `json:"rate_count"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	ChunkCount    int    `json:"chunk_count"`
}

type importChunk struct {
	currency string
	start    time.Time
	end      time.Time
}

func Open(configPath string) (*Service, error) {
	runtimeConfig, err := LoadRuntimeConfig(configPath)
	if err != nil {
		return nil, err
	}
	backend, err := openBackend(runtimeConfig.App)
	if err != nil {
		return nil, err
	}
	return &Service{
		configPath: runtimeConfig.Path,
		config:     runtimeConfig.App,
		backend:    backend,
	}, nil
}

func openBackend(cfg models.AppConfig) (*backendState, error) {
	store, err := cache.NewFileStore(cfg.CachePath)
	if err != nil {
		return nil, err
	}
	client := nbp.NewClient(nbp.ClientConfig{
		Timeout:         time.Duration(cfg.TimeoutSeconds) * time.Second,
		RetryCount:      cfg.RetryCount,
		MaxLookbackDays: cfg.MaxLookbackDays,
		Verbose:         cfg.Verbose,
		LogPath:         filepath.Join(filepath.Dir(cfg.CachePath), "nbp-client.log"),
	})
	return &backendState{
		store:    store,
		provider: nbp.NewService(client, store),
		client:   client,
	}, nil
}

func (s *Service) GetCurrencies(ctx context.Context) ([]models.Currency, error) {
	backend, cfg, _, err := s.acquireBackend()
	if err != nil {
		return nil, err
	}
	defer backend.active.Done()
	timedContext, cancel := context.WithTimeout(ctx, defaultTimeout(cfg.TimeoutSeconds))
	defer cancel()
	return backend.provider.GetCurrencies(timedContext)
}

func (s *Service) Dashboard() (Dashboard, error) {
	backend, cfg, configPath, err := s.acquireBackend()
	if err != nil {
		return Dashboard{}, err
	}
	defer backend.active.Done()
	info, err := backend.store.Info()
	if err != nil {
		return Dashboard{}, err
	}
	stats, err := backend.store.ListCurrencyStats()
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{
		ConfigPath: configPath,
		Config:     cfg,
		Cache:      info,
		Currencies: stats,
	}, nil
}

func (s *Service) Convert(ctx context.Context, request ConvertRequest) (ConversionResult, error) {
	code := strings.ToUpper(strings.TrimSpace(request.Currency))
	if !currencyPattern.MatchString(code) {
		return ConversionResult{}, fmt.Errorf("niepoprawny kod waluty: %s", request.Currency)
	}
	if math.IsNaN(request.Amount) || math.IsInf(request.Amount, 0) || request.Amount < 0 {
		return ConversionResult{}, fmt.Errorf("kwota musi być nieujemną liczbą skończoną")
	}
	date, err := ParseDate(request.Date)
	if err != nil {
		return ConversionResult{}, err
	}
	direction := strings.ToLower(strings.TrimSpace(request.Direction))
	if direction != "pln_to_foreign" && direction != "foreign_to_pln" {
		return ConversionResult{}, fmt.Errorf("niepoprawny kierunek przeliczenia")
	}

	backend, cfg, _, err := s.acquireBackend()
	if err != nil {
		return ConversionResult{}, err
	}
	timedContext, cancel := context.WithTimeout(ctx, defaultTimeout(cfg.TimeoutSeconds))
	rate, err := backend.provider.GetRate(timedContext, code, date)
	cancel()
	backend.active.Done()
	if err != nil {
		return ConversionResult{}, err
	}
	if math.IsNaN(rate.Mid) || math.IsInf(rate.Mid, 0) || rate.Mid <= 0 {
		return ConversionResult{}, fmt.Errorf("otrzymano niepoprawny kurs dla waluty %s", code)
	}

	result := ConversionResult{
		SourceAmount:   request.Amount,
		SourceCurrency: "PLN",
		TargetAmount:   request.Amount / rate.Mid,
		TargetCurrency: code,
		Rate:           rate,
	}
	if direction == "foreign_to_pln" {
		result.SourceCurrency = code
		result.TargetCurrency = "PLN"
		result.TargetAmount = request.Amount * rate.Mid
	}
	result.TargetAmount = math.Round(result.TargetAmount*currencyPrecision) / currencyPrecision
	if err := s.persistDate("converter", request.Date); err != nil {
		return ConversionResult{}, err
	}
	return result, nil
}

func (s *Service) ImportRates(ctx context.Context, request ImportRequest, progress func(ImportProgress)) (ImportSummary, error) {
	currencies, err := NormalizeCurrencies(request.Currencies)
	if err != nil {
		return ImportSummary{}, err
	}
	startDate, err := ParseDate(request.StartDate)
	if err != nil {
		return ImportSummary{}, err
	}
	endDate, err := ParseDate(request.EndDate)
	if err != nil {
		return ImportSummary{}, err
	}
	if endDate.Before(startDate) {
		return ImportSummary{}, fmt.Errorf("data końcowa nie może być wcześniejsza niż początkowa")
	}
	chunks := buildImportChunks(currencies, startDate, endDate)
	if err := s.persistDate("import", request.StartDate); err != nil {
		return ImportSummary{}, err
	}

	backend, cfg, _, err := s.acquireBackend()
	if err != nil {
		return ImportSummary{}, err
	}
	defer backend.active.Done()
	summary := ImportSummary{
		CurrencyCount: len(currencies),
		StartDate:     request.StartDate,
		EndDate:       request.EndDate,
		ChunkCount:    len(chunks),
	}
	for index, chunk := range chunks {
		if err := ctx.Err(); err != nil {
			return ImportSummary{}, err
		}
		chunkCtx, cancel := context.WithTimeout(ctx, defaultTimeout(cfg.TimeoutSeconds))
		count, importErr := backend.provider.ImportRateRangeChunk(chunkCtx, chunk.currency, chunk.start, chunk.end)
		cancel()
		if importErr != nil {
			return ImportSummary{}, fmt.Errorf("%s (%s - %s): %w", chunk.currency, chunk.start.Format("2006-01-02"), chunk.end.Format("2006-01-02"), importErr)
		}
		summary.RateCount += count
		if progress != nil {
			progress(ImportProgress{
				CompletedChunks: index + 1,
				TotalChunks:     len(chunks),
				Currency:        chunk.currency,
				ChunkStart:      chunk.start.Format("2006-01-02"),
				ChunkEnd:        chunk.end.Format("2006-01-02"),
				RateCount:       summary.RateCount,
			})
		}
	}
	return summary, nil
}

func (s *Service) CurrencyHistory(currency string, limit int) ([]cache.CurrencyHistoryEntry, error) {
	code := strings.ToUpper(strings.TrimSpace(currency))
	if !currencyPattern.MatchString(code) {
		return nil, fmt.Errorf("niepoprawny kod waluty: %s", currency)
	}
	if limit == 0 {
		limit = 120
	}
	backend, _, _, err := s.acquireBackend()
	if err != nil {
		return nil, err
	}
	defer backend.active.Done()
	return backend.store.ListCurrencyHistory(code, limit)
}

func (s *Service) ClearCache() error {
	backend, _, _, err := s.acquireBackend()
	if err != nil {
		return err
	}
	defer backend.active.Done()
	return backend.store.Clear()
}

func (s *Service) UpdateSettings(cfg models.AppConfig) (Dashboard, error) {
	s.reconfigureMu.Lock()
	defer s.reconfigureMu.Unlock()
	if err := validateSettings(cfg); err != nil {
		return Dashboard{}, err
	}
	cfg.Normalize()
	newBackend, err := openBackend(cfg)
	if err != nil {
		return Dashboard{}, err
	}
	s.mu.Lock()
	cfg.LastConverterDate = s.config.LastConverterDate
	cfg.LastFromDate = s.config.LastFromDate
	if err := SaveConfigAtPath(s.configPath, cfg); err != nil {
		s.mu.Unlock()
		_ = newBackend.close()
		return Dashboard{}, err
	}

	oldBackend := s.backend
	s.config = cfg
	s.backend = newBackend
	s.mu.Unlock()

	if oldBackend != nil {
		oldBackend.active.Wait()
		_ = oldBackend.close()
	}
	return s.Dashboard()
}

func (s *Service) Close() error {
	s.reconfigureMu.Lock()
	defer s.reconfigureMu.Unlock()
	s.mu.Lock()
	backend := s.backend
	s.backend = nil
	s.mu.Unlock()
	if backend == nil {
		return nil
	}
	backend.active.Wait()
	return backend.close()
}

func (s *Service) persistDate(kind, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if kind == "converter" {
		s.config.LastConverterDate = value
	} else {
		s.config.LastFromDate = value
	}
	return SaveConfigAtPath(s.configPath, s.config)
}

func (s *Service) acquireBackend() (*backendState, models.AppConfig, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backend == nil {
		return nil, models.AppConfig{}, "", errors.New("serwis aplikacji jest zamknięty")
	}
	s.backend.active.Add(1)
	return s.backend, s.config, s.configPath, nil
}

func (b *backendState) close() error {
	var clientErr, storeErr error
	if b.client != nil {
		clientErr = b.client.Close()
	}
	if b.store != nil {
		storeErr = b.store.Close()
	}
	return errors.Join(clientErr, storeErr)
}

func buildImportChunks(currencies []string, startDate, endDate time.Time) []importChunk {
	chunks := make([]importChunk, 0, len(currencies))
	for _, currency := range currencies {
		for chunkStart := startDate; !chunkStart.After(endDate); chunkStart = chunkStart.AddDate(0, 0, importChunkDays+1) {
			chunkEnd := chunkStart.AddDate(0, 0, importChunkDays)
			if chunkEnd.After(endDate) {
				chunkEnd = endDate
			}
			chunks = append(chunks, importChunk{currency: currency, start: chunkStart, end: chunkEnd})
		}
	}
	return chunks
}

func validateSettings(cfg models.AppConfig) error {
	if strings.TrimSpace(cfg.CachePath) == "" {
		return fmt.Errorf("ścieżka bazy nie może być pusta")
	}
	if cfg.TimeoutSeconds < 1 || cfg.TimeoutSeconds > 300 {
		return fmt.Errorf("limit czasu musi mieścić się w zakresie 1-300 sekund")
	}
	if cfg.RetryCount < 0 || cfg.RetryCount > 10 {
		return fmt.Errorf("liczba ponowień musi mieścić się w zakresie 0-10")
	}
	if cfg.MaxLookbackDays < 1 || cfg.MaxLookbackDays > 3660 {
		return fmt.Errorf("zakres wyszukiwania musi mieścić się w zakresie 1-3660 dni")
	}
	return nil
}

func defaultTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		seconds = models.DefaultTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func HumanizeError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.Canceled):
		return "operacja została anulowana"
	case errors.Is(err, cache.ErrCorruptedCache):
		return "plik bazy jest uszkodzony. Wyczyść bazę w aplikacji lub wybierz inny plik."
	case errors.Is(err, nbp.ErrNoData):
		return "brak danych kursowych dla podanej daty i waluty"
	case errors.Is(err, nbp.ErrTimeout), errors.Is(err, context.DeadlineExceeded):
		return "przekroczono limit czasu połączenia z API NBP"
	case errors.Is(err, nbp.ErrConnection):
		return "brak połączenia z API NBP. Sprawdź sieć i spróbuj ponownie"
	default:
		return strings.TrimSpace(err.Error())
	}
}
