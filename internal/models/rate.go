package models

import (
	"fmt"
	"math"
	"regexp"
	"time"
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

func ValidCurrencyCode(code string) bool { return currencyCodePattern.MatchString(code) }

// Validate rejects invalid values before they can enter the cache or a calculation.
func (r NBPRate) Validate(currency string) error {
	if !ValidCurrencyCode(currency) || r.Currency != currency {
		return fmt.Errorf("niezgodna waluta kursu: %q (oczekiwano %s)", r.Currency, currency)
	}
	date, err := time.Parse("2006-01-02", r.EffectiveRateDate)
	if err != nil || date.Format("2006-01-02") != r.EffectiveRateDate {
		return fmt.Errorf("niepoprawna data kursu: %q", r.EffectiveRateDate)
	}
	if math.IsNaN(r.Mid) || math.IsInf(r.Mid, 0) || r.Mid <= 0 {
		return fmt.Errorf("niepoprawny kurs dla waluty %s", currency)
	}
	return nil
}

type OutputFormat string

const (
	OutputText OutputFormat = "text"
	OutputJSON OutputFormat = "json"
)

type RateResult struct {
	Currency          string  `json:"currency"`
	RequestedDate     string  `json:"requested_date"`
	EffectiveRateDate string  `json:"effective_rate_date"`
	Mid               float64 `json:"mid"`
	TableNo           string  `json:"table_no,omitempty"`
	Source            string  `json:"source"`
}

type Currency struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type NBPRate struct {
	Currency          string
	EffectiveRateDate string
	Mid               float64
	TableNo           string
}

func (r NBPRate) ToResult(requestedDate, source string) RateResult {
	return RateResult{
		Currency:          r.Currency,
		RequestedDate:     requestedDate,
		EffectiveRateDate: r.EffectiveRateDate,
		Mid:               r.Mid,
		TableNo:           r.TableNo,
		Source:            source,
	}
}
