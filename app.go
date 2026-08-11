package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"kursomat/internal/appcore"
	"kursomat/internal/cache"
	"kursomat/internal/models"
)

const importProgressEvent = "import:progress"

type DesktopApp struct {
	mu            sync.RWMutex
	ctx           context.Context
	service       *appcore.Service
	initialiseErr error

	importMu     sync.Mutex
	importCancel context.CancelFunc
}

func NewDesktopApp() *DesktopApp {
	return &DesktopApp{}
}

func (a *DesktopApp) startup(ctx context.Context) {
	service, err := appcore.Open("")
	a.mu.Lock()
	a.ctx = ctx
	a.service = service
	a.initialiseErr = err
	a.mu.Unlock()
}

func (a *DesktopApp) shutdown(context.Context) {
	a.CancelImport()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.service != nil {
		_ = a.service.Close()
		a.service = nil
	}
}

func (a *DesktopApp) GetDashboard() (appcore.Dashboard, error) {
	service, _, err := a.backend()
	if err != nil {
		return appcore.Dashboard{}, err
	}
	result, err := service.Dashboard()
	return result, publicError(err)
}

func (a *DesktopApp) GetCurrencies() ([]models.Currency, error) {
	service, ctx, err := a.backend()
	if err != nil {
		return nil, err
	}
	result, err := service.GetCurrencies(ctx)
	return result, publicError(err)
}

func (a *DesktopApp) Convert(request appcore.ConvertRequest) (appcore.ConversionResult, error) {
	service, ctx, err := a.backend()
	if err != nil {
		return appcore.ConversionResult{}, err
	}
	result, err := service.Convert(ctx, request)
	return result, publicError(err)
}

func (a *DesktopApp) ImportRates(request appcore.ImportRequest) (appcore.ImportSummary, error) {
	service, appContext, err := a.backend()
	if err != nil {
		return appcore.ImportSummary{}, err
	}

	a.importMu.Lock()
	if a.importCancel != nil {
		a.importMu.Unlock()
		return appcore.ImportSummary{}, errors.New("import jest już uruchomiony")
	}
	ctx, cancel := context.WithCancel(appContext)
	a.importCancel = cancel
	a.importMu.Unlock()

	defer func() {
		cancel()
		a.importMu.Lock()
		a.importCancel = nil
		a.importMu.Unlock()
	}()

	result, err := service.ImportRates(ctx, request, func(progress appcore.ImportProgress) {
		runtime.EventsEmit(appContext, importProgressEvent, progress)
	})
	return result, publicError(err)
}

func (a *DesktopApp) CancelImport() bool {
	a.importMu.Lock()
	defer a.importMu.Unlock()
	if a.importCancel == nil {
		return false
	}
	a.importCancel()
	return true
}

func (a *DesktopApp) GetCurrencyHistory(currency string, limit int) ([]cache.CurrencyHistoryEntry, error) {
	service, _, err := a.backend()
	if err != nil {
		return nil, err
	}
	result, err := service.CurrencyHistory(currency, limit)
	return result, publicError(err)
}

func (a *DesktopApp) ExportCurrencyHistory(currency, format string) (string, error) {
	service, ctx, err := a.backend()
	if err != nil {
		return "", err
	}
	entries, err := service.CurrencyHistory(currency, -1)
	if err != nil {
		return "", publicError(err)
	}
	if len(entries) == 0 {
		return "", errors.New("brak notowań do wyeksportowania")
	}

	format = strings.ToLower(strings.TrimSpace(format))
	var filter runtime.FileFilter
	switch format {
	case "csv":
		filter = runtime.FileFilter{DisplayName: "Dane CSV (*.csv)", Pattern: "*.csv"}
	case "json":
		filter = runtime.FileFilter{DisplayName: "Dane JSON (*.json)", Pattern: "*.json"}
	default:
		return "", fmt.Errorf("nieobsługiwany format eksportu: %s", format)
	}

	dashboard, err := service.Dashboard()
	if err != nil {
		return "", publicError(err)
	}
	code := strings.ToUpper(strings.TrimSpace(currency))
	selection, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:                "Eksportuj historię kursów",
		DefaultDirectory:     filepath.Dir(dashboard.Config.CachePath),
		DefaultFilename:      fmt.Sprintf("kursomat-%s-%s.%s", code, time.Now().Format("20060102"), format),
		Filters:              []runtime.FileFilter{filter},
		CanCreateDirectories: true,
	})
	if err != nil {
		return "", publicError(err)
	}
	if selection == "" {
		return "", nil
	}
	if !strings.EqualFold(filepath.Ext(selection), "."+format) {
		selection += "." + format
	}
	if err := appcore.WriteHistoryExport(selection, format, code, entries); err != nil {
		return "", publicError(err)
	}
	return selection, nil
}

func (a *DesktopApp) ClearCache() error {
	if err := a.requireIdleImport("wyczyścić bazy"); err != nil {
		return err
	}
	service, _, err := a.backend()
	if err != nil {
		return err
	}
	return publicError(service.ClearCache())
}

func (a *DesktopApp) SaveSettings(config models.AppConfig) (appcore.Dashboard, error) {
	if err := a.requireIdleImport("zmienić ustawień"); err != nil {
		return appcore.Dashboard{}, err
	}
	service, _, err := a.backend()
	if err != nil {
		return appcore.Dashboard{}, err
	}
	result, err := service.UpdateSettings(config)
	return result, publicError(err)
}

func (a *DesktopApp) ChooseCachePath() (string, error) {
	_, ctx, err := a.backend()
	if err != nil {
		return "", err
	}
	dashboard, err := a.GetDashboard()
	if err != nil {
		return "", err
	}
	selection, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:            "Wybierz plik bazy Kursomatu",
		DefaultDirectory: filepath.Dir(dashboard.Config.CachePath),
		DefaultFilename:  filepath.Base(dashboard.Config.CachePath),
		Filters: []runtime.FileFilter{{
			DisplayName: "Baza SQLite (*.db)",
			Pattern:     "*.db",
		}},
		CanCreateDirectories: true,
	})
	if err != nil {
		return "", publicError(err)
	}
	return selection, nil
}

func (a *DesktopApp) backend() (*appcore.Service, context.Context, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.initialiseErr != nil {
		return nil, nil, publicError(a.initialiseErr)
	}
	if a.service == nil || a.ctx == nil {
		return nil, nil, errors.New("aplikacja nie jest jeszcze gotowa")
	}
	return a.service, a.ctx, nil
}

func (a *DesktopApp) requireIdleImport(action string) error {
	a.importMu.Lock()
	defer a.importMu.Unlock()
	if a.importCancel != nil {
		return fmt.Errorf("nie można %s podczas importu; najpierw zatrzymaj pobieranie", action)
	}
	return nil
}

func publicError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(appcore.HumanizeError(err))
}
