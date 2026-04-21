package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kursomat/internal/models"
)

var (
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
)

func ParseDate(raw string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, fmt.Errorf("niepoprawny format daty, użyj YYYY-MM-DD")
	}
	return parsed, nil
}

func ParseCurrencies(raw string) ([]string, error) {
	chunks := strings.Split(raw, ",")
	unique := make([]string, 0, len(chunks))
	seen := make(map[string]struct{}, len(chunks))

	for _, chunk := range chunks {
		code := strings.ToUpper(strings.TrimSpace(chunk))
		if code == "" {
			continue
		}
		if !currencyPattern.MatchString(code) {
			return nil, fmt.Errorf("niepoprawny kod waluty: %s", code)
		}
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		unique = append(unique, code)
	}
	if len(unique) == 0 {
		return nil, fmt.Errorf("podaj co najmniej jeden kod waluty")
	}
	return unique, nil
}

func ParseAmount(raw string) (float64, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if normalized == "" {
		return 0, fmt.Errorf("podaj kwotę do przeliczenia")
	}
	amount, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, fmt.Errorf("niepoprawna kwota: %s", raw)
	}
	if amount < 0 {
		return 0, fmt.Errorf("kwota nie może być ujemna")
	}
	return amount, nil
}

func ParseOutputFormat(raw string) (models.OutputFormat, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(models.OutputText):
		return models.OutputText, nil
	case string(models.OutputJSON):
		return models.OutputJSON, nil
	default:
		return "", fmt.Errorf("nieobsługiwany format wyjścia: %s (dozwolone: text, json)", raw)
	}
}
