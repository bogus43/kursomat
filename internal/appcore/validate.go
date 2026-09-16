package appcore

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"kursomat/internal/models"
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

func ParseDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return time.Time{}, fmt.Errorf("niepoprawna data %q; użyj formatu RRRR-MM-DD", raw)
	}
	if value < "2002-01-02" || value > models.NBPToday() {
		return time.Time{}, fmt.Errorf("data musi mieścić się między 2002-01-02 a bieżącą datą NBP")
	}
	return parsed, nil
}

func NormalizeCurrencies(currencies []string) ([]string, error) {
	seen := make(map[string]bool, len(currencies))
	result := make([]string, 0, len(currencies))
	for _, raw := range currencies {
		code := strings.ToUpper(strings.TrimSpace(raw))
		if !currencyPattern.MatchString(code) {
			return nil, fmt.Errorf("niepoprawny kod waluty: %s", raw)
		}
		if !seen[code] {
			seen[code] = true
			result = append(result, code)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("wybierz co najmniej jedną walutę")
	}
	return result, nil
}
