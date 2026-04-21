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
