package cli

import (
	"testing"

	"kursomat/internal/models"
)

func TestParseCurrencies(t *testing.T) {
	t.Parallel()

	valid, err := ParseCurrencies("usd, eur,CHF,usd")
	if err != nil {
		t.Fatalf("expected valid currencies, got error: %v", err)
	}
	if len(valid) != 3 {
		t.Fatalf("expected 3 unique currencies, got %d", len(valid))
	}
	if valid[0] != "USD" || valid[1] != "EUR" || valid[2] != "CHF" {
		t.Fatalf("unexpected currencies: %#v", valid)
	}

	if _, err := ParseCurrencies("US"); err == nil {
		t.Fatalf("expected invalid currency code error")
	}
}

func TestParseDate(t *testing.T) {
	t.Parallel()

	if _, err := ParseDate("2026-04-14"); err != nil {
		t.Fatalf("expected valid date, got error: %v", err)
	}
	if _, err := ParseDate("14-04-2026"); err == nil {
		t.Fatalf("expected invalid date format error")
	}
}

func TestParseOutputFormat(t *testing.T) {
	t.Parallel()

	got, err := ParseOutputFormat("json")
	if err != nil {
		t.Fatalf("expected json output format, got error: %v", err)
	}
	if got != models.OutputJSON {
		t.Fatalf("expected %q, got %q", models.OutputJSON, got)
	}
	if _, err := ParseOutputFormat("xml"); err == nil {
		t.Fatalf("expected unsupported output format error")
	}
}

func TestParseAmount(t *testing.T) {
	t.Parallel()

	got, err := ParseAmount("123,45")
	if err != nil {
		t.Fatalf("expected valid amount, got error: %v", err)
	}
	if got != 123.45 {
		t.Fatalf("expected 123.45, got %v", got)
	}
	if _, err := ParseAmount("-1"); err == nil {
		t.Fatalf("expected negative amount error")
	}
}
