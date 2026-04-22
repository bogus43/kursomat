package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"kursomat/internal/models"
)

func (a *App) registerCommonPFlags(fs *pflag.FlagSet, opts *configOptions) {
	fs.StringVar(&opts.configPath, "config", "", "Ścieżka do pliku konfiguracyjnego JSON")
	fs.StringVar(&opts.cachePath, "cache-path", "", "Nadpisanie ścieżki cache")
	fs.IntVar(&opts.timeoutSeconds, "timeout", -1, "Timeout żądania HTTP w sekundach")
	fs.IntVar(&opts.retryCount, "retry", -1, "Liczba retry dla błędów sieciowych")
	fs.IntVar(&opts.lookbackDays, "lookback-days", -1, "Maksymalny zakres cofnięcia daty przy szukaniu kursu")
	fs.BoolVar(&opts.verbose, "verbose", false, "Włącz dodatkowe logi diagnostyczne")
}

func (a *App) newRootCommand() *cobra.Command {
	commonOpts := &configOptions{}
	cobra.MousetrapHelpText = ""

	root := &cobra.Command{
		Use:           "kursomat",
		Short:         "Pełnoekranowa aplikacja terminalowa do kursów walut NBP",
		Long:          "Kursomat uruchamia domyślnie pełnoekranowy interfejs terminalowy. Dodatkowe komendy służą do szybkiego pobierania kursów i zarządzania lokalnym cache.",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeCfg, err := a.prepareRuntimeConfig(*commonOpts)
			if err != nil {
				return err
			}
			return a.executeTUI(runtimeCfg)
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetOut(a.out)
	root.SetErr(a.err)
	a.registerCommonPFlags(root.PersistentFlags(), commonOpts)

	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Uruchom pełnoekranowy interfejs terminalowy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeCfg, err := a.prepareRuntimeConfig(*commonOpts)
			if err != nil {
				return err
			}
			return a.executeTUI(runtimeCfg)
		},
	}

	var (
		currencyInput string
		dateInput     string
		outputInput   string
	)
	rateCmd := &cobra.Command{
		Use:   "rate",
		Short: "Pobierz kurs dla jednej lub wielu walut",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeCfg, err := a.prepareRuntimeConfig(*commonOpts)
			if err != nil {
				return err
			}

			currencies, err := ParseCurrencies(currencyInput)
			if err != nil {
				return err
			}
			requestedDate, err := ParseDate(dateInput)
			if err != nil {
				return err
			}
			outputFormat, err := ParseOutputFormat(outputInput)
			if err != nil {
				return err
			}
			return a.executeRate(runtimeCfg, currencies, requestedDate, outputFormat)
		},
	}
	rateCmd.Flags().StringVar(&currencyInput, "currency", "", "Kod waluty lub lista walut oddzielona przecinkiem (np. USD,EUR)")
	rateCmd.Flags().StringVar(&dateInput, "date", "", "Data w formacie YYYY-MM-DD")
	rateCmd.Flags().StringVar(&outputInput, "output", string(models.OutputText), "Format wyjścia: text | json")

	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Operacje na lokalnym cache SQLite",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd.Help()
			return fmt.Errorf("podaj podkomendę cache")
		},
	}

	cacheInfoCmd := &cobra.Command{
		Use:   "info",
		Short: "Pokaż statystyki lokalnego cache",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeCfg, err := a.prepareRuntimeConfig(*commonOpts)
			if err != nil {
				return err
			}
			return a.executeCacheInfo(runtimeCfg)
		},
	}

	cacheClearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Wyczyść lokalny cache",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeCfg, err := a.prepareRuntimeConfig(*commonOpts)
			if err != nil {
				return err
			}
			return a.executeCacheClear(runtimeCfg)
		},
	}

	cacheCmd.AddCommand(cacheInfoCmd, cacheClearCmd)
	root.AddCommand(tuiCmd, rateCmd, cacheCmd)
	return root
}
