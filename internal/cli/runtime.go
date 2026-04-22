package cli

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"kursomat/internal/cache"
	"kursomat/internal/models"
	"kursomat/internal/nbp"
)

type appRuntime struct {
	configPath string
	cfg        models.AppConfig
	store      cache.Store
	client     *nbp.Client
	service    *nbp.Service
}

func (a *App) prepareRuntimeConfig(opts configOptions) (RuntimeConfig, error) {
	runtimeCfg, err := LoadRuntimeConfig(opts.configPath)
	if err != nil {
		return RuntimeConfig{}, err
	}

	cfg := runtimeCfg.App
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

	runtimeCfg.App = cfg
	return runtimeCfg, nil
}

func (a *App) openRuntime(runtimeCfg RuntimeConfig, isTUI bool) (*appRuntime, error) {
	store, err := cache.NewFileStore(runtimeCfg.App.CachePath)
	if err != nil {
		return nil, err
	}

	client := nbp.NewClient(nbp.ClientConfig{
		Timeout:         time.Duration(runtimeCfg.App.TimeoutSeconds) * time.Second,
		RetryCount:      runtimeCfg.App.RetryCount,
		MaxLookbackDays: runtimeCfg.App.MaxLookbackDays,
		Verbose:         runtimeCfg.App.Verbose,
		IsTUI:           isTUI,
	})

	return &appRuntime{
		configPath: runtimeCfg.Path,
		cfg:        runtimeCfg.App,
		store:      store,
		client:     client,
		service:    nbp.NewService(client, store),
	}, nil
}

func (r *appRuntime) Close() error {
	var closeErr error
	if r.client != nil {
		if err := r.client.Close(); err != nil {
			closeErr = err
		}
	}
	if r.store != nil {
		if err := r.store.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func (a *App) executeRate(runtimeCfg RuntimeConfig, currencies []string, requestedDate time.Time, outputFormat models.OutputFormat) error {
	runtime, err := a.openRuntime(runtimeCfg, false)
	if err != nil {
		return err
	}
	defer runtime.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(runtime.cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	rates, err := runtime.service.GetRates(ctx, currencies, requestedDate)
	if err != nil {
		return err
	}
	if err := PrintRates(a.out, rates, outputFormat); err != nil {
		return fmt.Errorf("nie udało się wypisać wyników: %w", err)
	}
	return nil
}

func (a *App) executeTUI(runtimeCfg RuntimeConfig) error {
	runtime, err := a.openRuntime(runtimeCfg, true)
	if err != nil {
		return err
	}
	defer runtime.Close()

	model := newTUIModel(runtimeCfg.Path, runtime.cfg, runtime.service, runtime.store)
	program := tea.NewProgram(&model)
	if _, err := program.Run(); err != nil {
		return err
	}
	return nil
}

func (a *App) executeCacheClear(runtimeCfg RuntimeConfig) error {
	runtime, err := a.openRuntime(runtimeCfg, false)
	if err != nil {
		return err
	}
	defer runtime.Close()

	if err := runtime.store.Clear(); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "Cache został wyczyszczony.")
	return nil
}

func (a *App) executeCacheInfo(runtimeCfg RuntimeConfig) error {
	runtime, err := a.openRuntime(runtimeCfg, false)
	if err != nil {
		return err
	}
	defer runtime.Close()

	info, err := runtime.store.Info()
	if err != nil {
		return err
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
	return nil
}
