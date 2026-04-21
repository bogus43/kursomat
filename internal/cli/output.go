package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"kursomat/internal/models"
)

func PrintRates(w io.Writer, rates []models.RateResult, format models.OutputFormat) error {
	switch format {
	case models.OutputText:
		return printRatesText(w, rates)
	case models.OutputJSON:
		return printRatesJSON(w, rates)
	default:
		return fmt.Errorf("nieznany format wyjścia: %s", format)
	}
}

func printRatesText(w io.Writer, rates []models.RateResult) error {
	for i, rate := range rates {
		if _, err := fmt.Fprintf(
			w,
			"Waluta: %s\nData żądana: %s\nData kursu: %s\nKurs średni NBP: %.4f\nTabela: %s\nŹródło: %s\n",
			rate.Currency,
			rate.RequestedDate,
			rate.EffectiveRateDate,
			rate.Mid,
			emptyIfMissing(rate.TableNo),
			rate.Source,
		); err != nil {
			return err
		}
		if i < len(rates)-1 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}
	return nil
}

func printRatesJSON(w io.Writer, rates []models.RateResult) error {
	var payload any
	if len(rates) == 1 {
		payload = rates[0]
	} else {
		payload = rates
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("nie udało się zserializować wyniku do JSON: %w", err)
	}
	if _, err := fmt.Fprintln(w, string(data)); err != nil {
		return err
	}
	return nil
}

func emptyIfMissing(in string) string {
	if in == "" {
		return "-"
	}
	return in
}
