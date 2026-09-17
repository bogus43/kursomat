package appcore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"kursomat/internal/cache"
	"kursomat/internal/nbp"
)

func TestHumanizeError(t *testing.T) {
	if got := HumanizeError(nil); got != "" {
		t.Fatalf("nil error: %q", got)
	}
	for _, tc := range []struct {
		err  error
		want string
	}{
		{context.Canceled, "operacja została anulowana"},
		{cache.ErrCorruptedCache, "plik bazy jest uszkodzony. Wyczyść bazę w aplikacji lub wybierz inny plik."},
		{nbp.ErrNoData, "brak danych kursowych dla podanej daty i waluty"},
		{nbp.ErrTimeout, "przekroczono limit czasu połączenia z API NBP"},
		{context.DeadlineExceeded, "przekroczono limit czasu połączenia z API NBP"},
		{nbp.ErrConnection, "brak połączenia z API NBP. Sprawdź sieć i spróbuj ponownie"},
	} {
		for _, err := range []error{tc.err, fmt.Errorf("internal detail: %w", tc.err)} {
			if got := HumanizeError(err); got != tc.want {
				t.Errorf("%v: got %q, want %q", err, got, tc.want)
			}
		}
	}
	if got := HumanizeError(errors.New("  custom failure \n")); got != "custom failure" {
		t.Fatalf("fallback = %q", got)
	}
}
