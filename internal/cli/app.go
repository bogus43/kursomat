package cli

import (
	"errors"
	"fmt"
	"io"
	"kursomat/internal/cache"
	"kursomat/internal/nbp"
	"strings"
)

type App struct {
	out io.Writer
	err io.Writer
}

type configOptions struct {
	configPath     string
	cachePath      string
	timeoutSeconds int
	retryCount     int
	lookbackDays   int
	verbose        bool
}

func NewApp(out, err io.Writer) *App {
	return &App{out: out, err: err}
}

func (a *App) Run(args []string) int {
	cmd := a.newRootCommand()
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		a.printError(err)
		return 1
	}
	return 0
}

func (a *App) printError(err error) {
	fmt.Fprintf(a.err, "Błąd: %s\n", humanizeError(err))
}

func humanizeError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, cache.ErrCorruptedCache):
		return "plik cache jest uszkodzony. Usuń cache poleceniem `kursomat cache clear`."
	case errors.Is(err, nbp.ErrNoData):
		return "brak danych kursowych dla podanej daty i waluty."
	case errors.Is(err, nbp.ErrTimeout):
		return "przekroczono limit czasu połączenia z API NBP."
	case errors.Is(err, nbp.ErrConnection):
		return "brak połączenia z API NBP. Sprawdź sieć i spróbuj ponownie."
	default:
		return strings.TrimSpace(err.Error())
	}
}

func orDash(in string) string {
	if strings.TrimSpace(in) == "" {
		return "-"
	}
	return in
}
