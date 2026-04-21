package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"kursomat/internal/cache"
	"kursomat/internal/models"
	"kursomat/internal/nbp"
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

func (a *App) registerCommonFlags(fs *flag.FlagSet, opts *configOptions) {
	fs.StringVar(&opts.configPath, "config", "", "Ścieżka do pliku konfiguracyjnego JSON")
	fs.StringVar(&opts.cachePath, "cache-path", "", "Nadpisanie ścieżki cache")
	fs.IntVar(&opts.timeoutSeconds, "timeout", -1, "Timeout żądania HTTP w sekundach")
	fs.IntVar(&opts.retryCount, "retry", -1, "Liczba retry dla błędów sieciowych")
	fs.IntVar(&opts.lookbackDays, "lookback-days", -1, "Maksymalny zakres cofnięcia daty przy szukaniu kursu")
	fs.BoolVar(&opts.verbose, "verbose", false, "Włącz dodatkowe logi diagnostyczne")
}

func (a *App) prepareConfig(opts configOptions) (models.AppConfig, error) {
	cfg, err := LoadConfig(opts.configPath)
	if err != nil {
		return cfg, err
	}
	if opts.cachePath != "" {
		cfg.CachePath = opts.cachePath
	}
	if opts.timeoutSeconds > 0 {
		cfg.TimeoutSeconds = opts.timeoutSeconds
	}
	if opts.retryCount >= 0 {
		cfg.RetryCount = opts.retryCount
	}
	if opts.lookbackDays > 0 {
		cfg.MaxLookbackDays = opts.lookbackDays
	}
	if opts.verbose {
		cfg.Verbose = true
	}
	cfg.Normalize()
	return cfg, nil
}

func (a *App) Run(args []string) int {
	if len(args) == 0 {
		return a.runTUI(nil)
	}

	switch args[0] {
	case "tui":
		return a.runTUI(args[1:])
	case "rate":
		return a.runRate(args[1:])
	case "cache":
		return a.runCache(args[1:])
	case "help", "--help", "-h":
		a.printRootUsage()
		return 0
	default:
		fmt.Fprintf(a.err, "Nieznana komenda: %s\n\n", args[0])
		a.printRootUsage()
		return 1
	}
}

func (a *App) runRate(args []string) int {
	fs := flag.NewFlagSet("rate", flag.ContinueOnError)
	fs.SetOutput(a.err)

	var (
		currencyInput string
		dateInput     string
		outputInput   string
		opts          configOptions
	)

	fs.StringVar(&currencyInput, "currency", "", "Kod waluty lub lista walut oddzielona przecinkiem (np. USD,EUR)")
	fs.StringVar(&dateInput, "date", "", "Data w formacie YYYY-MM-DD")
	fs.StringVar(&outputInput, "output", string(models.OutputText), "Format wyjścia: text | json")
	a.registerCommonFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}

	cfg, err := a.prepareConfig(opts)
	if err != nil {
		a.printError(err)
		return 1
	}

	currencies, err := ParseCurrencies(currencyInput)
	if err != nil {
		a.printError(err)
		return 1
	}
	requestedDate, err := ParseDate(dateInput)
	if err != nil {
		a.printError(err)
		return 1
	}
	outputFormat, err := ParseOutputFormat(outputInput)
	if err != nil {
		a.printError(err)
		return 1
	}

	store, err := cache.NewFileStore(cfg.CachePath)
	if err != nil {
		a.printError(err)
		return 1
	}
	defer store.Close()
	client := nbp.NewClient(nbp.ClientConfig{
		Timeout:         time.Duration(cfg.TimeoutSeconds) * time.Second,
		RetryCount:      cfg.RetryCount,
		MaxLookbackDays: cfg.MaxLookbackDays,
		Verbose:         cfg.Verbose,
	})
	service := nbp.NewService(client, store)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	rates, err := service.GetRates(ctx, currencies, requestedDate)
	if err != nil {
		a.printError(err)
		return 1
	}
	if err := PrintRates(a.out, rates, outputFormat); err != nil {
		a.printError(err)
		return 1
	}
	return 0
}

func (a *App) runTUI(args []string) int {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(a.err)

	var opts configOptions
	a.registerCommonFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}

	cfg, err := a.prepareConfig(opts)
	if err != nil {
		a.printError(err)
		return 1
	}

	store, err := cache.NewFileStore(cfg.CachePath)
	if err != nil {
		a.printError(err)
		return 1
	}
	defer store.Close()
	client := nbp.NewClient(nbp.ClientConfig{
		Timeout:         time.Duration(cfg.TimeoutSeconds) * time.Second,
		RetryCount:      cfg.RetryCount,
		MaxLookbackDays: cfg.MaxLookbackDays,
		Verbose:         cfg.Verbose,
		IsTUI:           true,
	})
	service := nbp.NewService(client, store)

	model := newTUIModel(cfg, service, store)
	program := tea.NewProgram(&model)
	if _, err := program.Run(); err != nil {
		a.printError(err)
		return 1
	}
	return 0
}

func (a *App) runCache(args []string) int {
	if len(args) == 0 {
		a.printCacheUsage()
		return 1
	}

	switch args[0] {
	case "clear":
		return a.runCacheClear(args[1:])
	case "info":
		return a.runCacheInfo(args[1:])
	case "help", "--help", "-h":
		a.printCacheUsage()
		return 0
	default:
		fmt.Fprintf(a.err, "Nieznana komenda cache: %s\n\n", args[0])
		a.printCacheUsage()
		return 1
	}
}

func (a *App) runCacheClear(args []string) int {
	fs := flag.NewFlagSet("cache clear", flag.ContinueOnError)
	fs.SetOutput(a.err)
	var opts configOptions
	fs.StringVar(&opts.configPath, "config", "", "Ścieżka do pliku konfiguracyjnego JSON")
	fs.StringVar(&opts.cachePath, "cache-path", "", "Nadpisanie ścieżki cache")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}

	cfg, err := a.prepareConfig(opts)
	if err != nil {
		a.printError(err)
		return 1
	}

	store, err := cache.NewFileStore(cfg.CachePath)
	if err != nil {
		a.printError(err)
		return 1
	}
	defer store.Close()
	if err := store.Clear(); err != nil {
		a.printError(err)
		return 1
	}
	fmt.Fprintln(a.out, "Cache został wyczyszczony.")
	return 0
}

func (a *App) runCacheInfo(args []string) int {
	fs := flag.NewFlagSet("cache info", flag.ContinueOnError)
	fs.SetOutput(a.err)
	var opts configOptions
	fs.StringVar(&opts.configPath, "config", "", "Ścieżka do pliku konfiguracyjnego JSON")
	fs.StringVar(&opts.cachePath, "cache-path", "", "Nadpisanie ścieżki cache")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}

	cfg, err := a.prepareConfig(opts)
	if err != nil {
		a.printError(err)
		return 1
	}

	store, err := cache.NewFileStore(cfg.CachePath)
	if err != nil {
		a.printError(err)
		return 1
	}
	defer store.Close()

	info, err := store.Info()
	if err != nil {
		a.printError(err)
		return 1
	}
	fmt.Fprintf(
		a.out,
		"Ścieżka: %s\nLiczba wpisów kursów: %d\nLiczba mapowań zapytań: %d\nLiczba walut: %d\nRozmiar pliku: %d B\nOstatni zapis: %s\n",
		info.Path,
		info.Entries,
		info.QueryMappings,
		info.CurrencyCount,
		info.SizeBytes,
		orDash(info.LastSavedAt),
	)
	return 0
}

func (a *App) printRootUsage() {
	fmt.Fprintln(a.out, "Kursownik NBP")
	fmt.Fprintln(a.out, "")
	fmt.Fprintln(a.out, "Użycie:")
	fmt.Fprintln(a.out, "  kursownik-nbp tui")
	fmt.Fprintln(a.out, "  kursownik-nbp rate --currency USD --date 2026-04-14")
	fmt.Fprintln(a.out, "  kursownik-nbp rate --currency USD,EUR,CHF --date 2026-04-14 --output json")
	fmt.Fprintln(a.out, "  kursownik-nbp cache clear")
	fmt.Fprintln(a.out, "  kursownik-nbp cache info")
}

func (a *App) printCacheUsage() {
	fmt.Fprintln(a.out, "Użycie:")
	fmt.Fprintln(a.out, "  kursownik-nbp cache clear")
	fmt.Fprintln(a.out, "  kursownik-nbp cache info")
}

func (a *App) printError(err error) {
	fmt.Fprintf(a.err, "Błąd: %s\n", humanizeError(err))
}

func humanizeError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, cache.ErrCorruptedCache):
		return "plik cache jest uszkodzony. Usuń cache poleceniem `kursownik-nbp cache clear`."
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
