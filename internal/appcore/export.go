package appcore

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"kursomat/internal/cache"
)

type HistoryExport struct {
	Currency string                       `json:"currency"`
	Entries  []cache.CurrencyHistoryEntry `json:"entries"`
}

func WriteHistoryExport(path, format, currency string, entries []cache.CurrencyHistoryEntry) error {
	code := strings.ToUpper(strings.TrimSpace(currency))
	if !currencyPattern.MatchString(code) {
		return fmt.Errorf("niepoprawny kod waluty: %s", currency)
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("ścieżka eksportu nie może być pusta")
	}

	var (
		data []byte
		err  error
	)
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv":
		data, err = encodeHistoryCSV(code, entries)
	case "json":
		data, err = json.MarshalIndent(HistoryExport{Currency: code, Entries: entries}, "", "  ")
		if err == nil {
			data = append(data, '\n')
		}
	default:
		return fmt.Errorf("nieobsługiwany format eksportu: %s", format)
	}
	if err != nil {
		return fmt.Errorf("nie udało się przygotować eksportu: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("nie udało się zapisać eksportu: %w", err)
	}
	return nil
}

func encodeHistoryCSV(currency string, entries []cache.CurrencyHistoryEntry) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.Write([]byte{0xef, 0xbb, 0xbf})
	writer := csv.NewWriter(&buffer)
	writer.Comma = ';'
	if err := writer.Write([]string{"currency", "effective_rate_date", "mid", "table_no"}); err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if err := writer.Write([]string{
			currency,
			entry.EffectiveRateDate,
			strconv.FormatFloat(entry.Mid, 'f', -1, 64),
			entry.TableNo,
		}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
