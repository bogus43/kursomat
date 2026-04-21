package nbp

import (
	"context"
	"fmt"
	"time"

	"kursomat/internal/cache"
	"kursomat/internal/models"
)

type Service struct {
	client *Client
	cache  cache.Store
}

type PrefetchSummary struct {
	CurrencyCount int
	RateCount     int
	StartDate     string
	EndDate       string
}

func NewService(client *Client, store cache.Store) *Service {
	return &Service{client: client, cache: store}
}

func (s *Service) GetCurrencies(ctx context.Context) ([]models.Currency, error) {
	if s.cache != nil {
		currencies, found, err := s.cache.GetCurrencies()
		if err != nil {
			return nil, err
		}
		if found {
			return currencies, nil
		}
	}

	currencies, err := s.client.GetCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		if err := s.cache.StoreCurrencies(currencies); err != nil {
			return nil, err
		}
	}
	return currencies, nil
}

func (s *Service) GetRates(ctx context.Context, currencies []string, requestedDate time.Time) ([]models.RateResult, error) {
	results := make([]models.RateResult, 0, len(currencies))
	for _, currency := range currencies {
		result, err := s.GetRate(ctx, currency, requestedDate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", currency, err)
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) GetRate(ctx context.Context, currency string, requestedDate time.Time) (models.RateResult, error) {
	requestedDateStr := requestedDate.Format("2006-01-02")

	if s.cache != nil {
		cached, found, err := s.cache.GetByQuery(currency, requestedDateStr)
		if err != nil {
			return models.RateResult{}, err
		}
		if found {
			return cached, nil
		}

		historical, found, err := s.cache.GetLatestRate(currency, requestedDateStr)
		if err != nil {
			return models.RateResult{}, err
		}
		if found {
			if err := s.cache.StoreResolvedRate(currency, requestedDateStr, models.NBPRate{
				Currency:          historical.Currency,
				EffectiveRateDate: historical.EffectiveRateDate,
				Mid:               historical.Mid,
				TableNo:           historical.TableNo,
			}); err != nil {
				return models.RateResult{}, err
			}
			return historical, nil
		}
	}

	rate, err := s.client.GetRateOnOrBefore(ctx, currency, requestedDate)
	if err != nil {
		return models.RateResult{}, err
	}
	if s.cache != nil {
		if err := s.cache.StoreResolvedRate(currency, requestedDateStr, rate); err != nil {
			return models.RateResult{}, err
		}
	}
	return rate.ToResult(requestedDateStr, "NBP API"), nil
}

func (s *Service) ImportRateRangeChunk(ctx context.Context, currency string, startDate, endDate time.Time) (int, error) {
	if endDate.Before(startDate) {
		return 0, fmt.Errorf("data końcowa nie może być wcześniejsza niż początkowa")
	}

	rates, err := s.client.GetRatesInRange(ctx, currency, startDate, endDate)
	if err != nil {
		if err == ErrNoData {
			return 0, nil
		}
		return 0, fmt.Errorf("%s: %w", currency, err)
	}

	if s.cache != nil {
		if err := s.cache.StoreHistoricalRates(currency, rates); err != nil {
			return 0, err
		}
	}

	return len(rates), nil
}

func (s *Service) PrefetchRates(ctx context.Context, currencies []string, startDate, endDate time.Time) (PrefetchSummary, error) {
	if endDate.Before(startDate) {
		return PrefetchSummary{}, fmt.Errorf("data końcowa nie może być wcześniejsza niż początkowa")
	}
	if len(currencies) == 0 {
		return PrefetchSummary{}, fmt.Errorf("wybierz co najmniej jedną walutę")
	}

	const chunkDays = 90
	summary := PrefetchSummary{
		StartDate: startDate.Format("2006-01-02"),
		EndDate:   endDate.Format("2006-01-02"),
	}

	for _, currency := range currencies {
		summary.CurrencyCount++
		for chunkStart := startDate; !chunkStart.After(endDate); chunkStart = chunkStart.AddDate(0, 0, chunkDays+1) {
			chunkEnd := chunkStart.AddDate(0, 0, chunkDays)
			if chunkEnd.After(endDate) {
				chunkEnd = endDate
			}
			count, err := s.ImportRateRangeChunk(ctx, currency, chunkStart, chunkEnd)
			if err != nil {
				return PrefetchSummary{}, err
			}
			summary.RateCount += count
		}
	}

	return summary, nil
}
