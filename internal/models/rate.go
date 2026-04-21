package models

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
