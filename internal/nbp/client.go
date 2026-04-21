package nbp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"kursomat/internal/models"
)

type ClientConfig struct {
	BaseURL         string
	Timeout         time.Duration
	RetryCount      int
	MaxLookbackDays int
	Verbose         bool
	HTTPClient      *http.Client
}

type Client struct {
	baseURL         string
	httpClient      *http.Client
	retryCount      int
	maxLookbackDays int
	verbose         bool
	logger          *log.Logger
}

type rateRangeResponse struct {
	Table    string       `json:"table"`
	Currency string       `json:"currency"`
	Code     string       `json:"code"`
	Rates    []nbpRateRow `json:"rates"`
}

type tableAResponse []nbpTable

type nbpTable struct {
	Table string         `json:"table"`
	No    string         `json:"no"`
	Rates []nbpTableRate `json:"rates"`
}

type nbpRateRow struct {
	No            string  `json:"no"`
	EffectiveDate string  `json:"effectiveDate"`
	Mid           float64 `json:"mid"`
}

type nbpTableRate struct {
	Currency string  `json:"currency"`
	Code     string  `json:"code"`
	Mid      float64 `json:"mid"`
}

func NewClient(cfg ClientConfig) *Client {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.nbp.pl/api"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	} else if httpClient.Timeout <= 0 {
		httpClient.Timeout = timeout
	}
	retryCount := cfg.RetryCount
	if retryCount < 0 {
		retryCount = 0
	}
	maxLookback := cfg.MaxLookbackDays
	if maxLookback <= 0 {
		maxLookback = 92
	}

	return &Client{
		baseURL:         strings.TrimRight(baseURL, "/"),
		httpClient:      httpClient,
		retryCount:      retryCount,
		maxLookbackDays: maxLookback,
		verbose:         cfg.Verbose,
		logger:          log.New(os.Stderr, "kursownik-nbp ", log.LstdFlags),
	}
}

func (c *Client) GetRateOnOrBefore(ctx context.Context, currency string, requestedDate time.Time) (models.NBPRate, error) {
	endDate := requestedDate.Format("2006-01-02")
	startDate := requestedDate.AddDate(0, 0, -c.maxLookbackDays).Format("2006-01-02")
	endpoint := fmt.Sprintf(
		"%s/exchangerates/rates/A/%s/%s/%s/?format=json",
		c.baseURL,
		strings.ToUpper(currency),
		startDate,
		endDate,
	)

	var payload rateRangeResponse
	if err := c.doJSON(ctx, endpoint, &payload); err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.NBPRate{}, ErrNoData
		}
		return models.NBPRate{}, err
	}
	return pickLatestRate(payload, strings.ToUpper(currency), requestedDate)
}

func (c *Client) GetCurrencies(ctx context.Context) ([]models.Currency, error) {
	endpoint := fmt.Sprintf("%s/exchangerates/tables/A/?format=json", c.baseURL)

	var payload tableAResponse
	if err := c.doJSON(ctx, endpoint, &payload); err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, ErrNoData
	}

	currencies := make([]models.Currency, 0, len(payload[0].Rates))
	for _, item := range payload[0].Rates {
		currencies = append(currencies, models.Currency{
			Code: item.Code,
			Name: item.Currency,
		})
	}
	return currencies, nil
}

func pickLatestRate(payload rateRangeResponse, currency string, requestedDate time.Time) (models.NBPRate, error) {
	if len(payload.Rates) == 0 {
		return models.NBPRate{}, ErrNoData
	}

	for i := len(payload.Rates) - 1; i >= 0; i-- {
		current := payload.Rates[i]
		rateDate, err := time.Parse("2006-01-02", current.EffectiveDate)
		if err != nil {
			return models.NBPRate{}, fmt.Errorf("nie udało się sparsować daty kursu z API: %w", err)
		}
		if !rateDate.After(requestedDate) {
			return models.NBPRate{
				Currency:          currency,
				EffectiveRateDate: current.EffectiveDate,
				Mid:               current.Mid,
				TableNo:           current.No,
			}, nil
		}
	}
	return models.NBPRate{}, ErrNoData
}

func parseRateRangeResponse(data []byte) (rateRangeResponse, error) {
	var payload rateRangeResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return rateRangeResponse{}, fmt.Errorf("nie udało się odczytać odpowiedzi API: %w", err)
	}
	return payload, nil
}

func (c *Client) doJSON(ctx context.Context, endpoint string, dst any) error {
	var lastErr error
	for attempt := 0; attempt <= c.retryCount; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * 200 * time.Millisecond
			if err := waitBackoff(ctx, backoff); err != nil {
				return mapNetworkError(err)
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return fmt.Errorf("nie udało się utworzyć zapytania do API NBP: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "kursownik-nbp/1.0")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = mapNetworkError(err)
			if c.verbose {
				c.logger.Printf("próba %d/%d nieudana: %v", attempt+1, c.retryCount+1, err)
			}
			if shouldRetryNetworkError(err) && attempt < c.retryCount {
				continue
			}
			return lastErr
		}

		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("nie udało się odczytać odpowiedzi API NBP: %w", readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("nie udało się zamknąć odpowiedzi HTTP: %w", closeErr)
		}

		if resp.StatusCode == http.StatusOK {
			if err := json.Unmarshal(body, dst); err != nil {
				return fmt.Errorf("nie udało się odczytać odpowiedzi API: %w", err)
			}
			return nil
		}
		if resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("API NBP chwilowo niedostępne (HTTP %d)", resp.StatusCode)
			if c.verbose {
				c.logger.Printf("próba %d/%d: %v", attempt+1, c.retryCount+1, lastErr)
			}
			if attempt < c.retryCount {
				continue
			}
			return lastErr
		}
		return fmt.Errorf("API NBP zwróciło błąd HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if lastErr == nil {
		lastErr = errors.New("nieznany błąd podczas zapytania do API NBP")
	}
	return lastErr
}

func waitBackoff(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func shouldRetryNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}

func mapNetworkError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && errors.Is(urlErr.Err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrTimeout
	}
	return ErrConnection
}
