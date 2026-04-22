package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"kursomat/internal/cache"
	"kursomat/internal/models"
	"kursomat/internal/nbp"
)

// ZGODNOŚĆ Z CLI-APP-DEV-INSTRUCTIONS:
// 1. Stos: Bubble Tea v2, Bubbles v2, Lip Gloss v2.
// 2. Nieblokujące Update: Wszystkie operacje I/O i sieciowe w tea.Cmd.
// 3. Responsywność: Resize w tea.WindowSizeMsg.
// 4. Logowanie: Client loguje do logs/nbp-client.log (charm.land/log/v2).

type conversionDirection int

const (
	directionPLNToForeign conversionDirection = iota
	directionForeignToPLN
)

type directionMode int

const (
	directionModeAuto directionMode = iota
	directionModeManual
)

type currenciesLoadedMsg struct {
	currencies []models.Currency
	err        error
}

type currencyClickedMsg struct {
	target string
	index  int
}

type mouseActionMsg struct {
	action string
}

type convertFinishedMsg struct {
	result       string
	direction    conversionDirection
	sourceAmount float64
	targetAmount float64
	err          error
}

type prefetchFinishedMsg struct {
	summary nbp.PrefetchSummary
	err     error
}

type prefetchChunkFinishedMsg struct {
	chunk     prefetchChunk
	rateCount int
	err       error
}

type cacheInfoLoadedMsg struct {
	info cache.Info
	err  error
}

type currencyStatsLoadedMsg struct {
	stats []cache.CurrencyStat
	err   error
}

type currencyHistoryLoadedMsg struct {
	currency string
	history  []cache.CurrencyHistoryEntry
	err      error
}

type cacheClearedMsg struct {
	err error
}

type configSavedMsg struct {
	err error
}

type converterCurrencyItem struct {
	code string
	name string
}

type cacheCurrencyItem struct {
	code     string
	name     string
	selected bool
}

type prefetchChunk struct {
	currency string
	start    time.Time
	end      time.Time
}

type amountField int

const (
	amountFieldPLN amountField = iota
	amountFieldForeign
)

type dbSortMode int

const (
	dbSortByCode dbSortMode = iota
	dbSortByRateCount
	dbSortByLastDate
)

type dbCurrencyItem struct {
	stat cache.CurrencyStat
}

type pickerTarget int

const (
	pickerTargetNone pickerTarget = iota
	pickerTargetCache
)

func (i dbCurrencyItem) Title() string {
	return fmt.Sprintf("%s  %s", i.stat.Code, i.stat.Name)
}

func (i dbCurrencyItem) Description() string {
	return fmt.Sprintf("kursów: %d | pierwszy: %s | ostatni: %s", i.stat.RateCount, orDash(i.stat.FirstDate), orDash(i.stat.LastDate))
}

func (i dbCurrencyItem) FilterValue() string {
	return i.stat.Code + " " + i.stat.Name
}

func (c converterCurrencyItem) Title() string {
	return strings.ToUpper(c.code) + "  " + c.name
}

func (c converterCurrencyItem) Description() string { return c.name }
func (c converterCurrencyItem) FilterValue() string { return c.code + " " + c.name }

func (c cacheCurrencyItem) Title() string {
	if c.selected {
		return "[x] " + strings.ToUpper(c.code) + "  " + c.name
	}
	return "[ ] " + strings.ToUpper(c.code) + "  " + c.name
}
func (c cacheCurrencyItem) Description() string { return c.name }
func (c cacheCurrencyItem) FilterValue() string { return c.code + " " + c.name }

type tuiKeyMap struct {
	PrevTab    key.Binding
	NextTab    key.Binding
	NextFocus  key.Binding
	PrevFocus  key.Binding
	Search     key.Binding
	ToggleDir  key.Binding
	ToggleAll  key.Binding
	Refresh    key.Binding
	ClearCache key.Binding
	ToggleHelp key.Binding
	Quit       key.Binding
}

func defaultTUIKeyMap() tuiKeyMap {
	return tuiKeyMap{
		PrevTab:    key.NewBinding(key.WithKeys("ctrl+shift+tab"), key.WithHelp("ctrl+shift+tab", "poprzednia zakładka")),
		NextTab:    key.NewBinding(key.WithKeys("ctrl+tab"), key.WithHelp("ctrl+tab", "następna zakładka")),
		NextFocus:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "następny fokus")),
		PrevFocus:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "poprzedni fokus")),
		Search:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "szukaj waluty")),
		ToggleDir:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "odwróć kierunek")),
		ToggleAll:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "wszystkie waluty")),
		Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "odśwież cache")),
		ClearCache: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "wyczyść cache")),
		ToggleHelp: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "pomoc")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "wyjście")),
	}
}

func (k tuiKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.PrevTab, k.NextTab, k.Search, k.ToggleDir, k.Quit}
}

func (k tuiKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.PrevTab, k.NextTab, k.NextFocus, k.PrevFocus},
		{k.Search, k.ToggleDir, k.Refresh, k.ClearCache},
		{k.ToggleDir, k.Refresh, k.ClearCache, k.ToggleHelp, k.Quit},
	}
}

type tuiModel struct {
	configPath string
	cfg        models.AppConfig
	service    *nbp.Service
	store      cache.Store

	keys     tuiKeyMap
	help     help.Model
	spinner  spinner.Model
	progress progress.Model

	width  int
	height int

	activeTab int
	focus     int

	currencyList       list.Model
	cacheCurrencyList  list.Model
	dbCurrencyList     list.Model
	plnAmountInput     textinput.Model
	foreignAmountInput textinput.Model
	dateInput          textinput.Model
	cacheFromInput     textinput.Model
	cacheToInput       textinput.Model
	dbFilterInput      textinput.Model

	resultsViewport viewport.Model
	cacheViewport   viewport.Model
	dbViewport      viewport.Model

	direction               conversionDirection
	directionMode           directionMode
	lastEditedAmount        amountField
	loading                 bool
	cacheBusy               bool
	allCacheCurrencies      bool
	currencyPickerOpen      bool
	currencyPickerTarget    pickerTarget
	prefetchChunks          []prefetchChunk
	prefetchDone            int
	prefetchSummary         nbp.PrefetchSummary
	lastError               string
	status                  string
	showHelp                bool
	currenciesReady         bool
	cacheInfo               cache.Info
	currencyStats           []cache.CurrencyStat
	filteredCurrencyStats   []cache.CurrencyStat
	selectedDBCurrency      string
	selectedDBHistory       []cache.CurrencyHistoryEntry
	dbSortMode              dbSortMode
	currencies              []models.Currency
	selectedCacheCurrencies map[string]bool
}

func newTUIModel(configPath string, cfg models.AppConfig, service *nbp.Service, store cache.Store) tuiModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	currencyList := list.New([]list.Item{}, delegate, 32, 12)
	currencyList.Title = "PICKER WALUT NBP"
	currencyList.SetShowHelp(false)
	currencyList.SetShowPagination(true)
	currencyList.SetShowStatusBar(true)
	currencyList.SetFilteringEnabled(true)
	currencyList.SetShowFilter(true)
	currencyList.DisableQuitKeybindings()
	currencyList.FilterInput.Placeholder = "Szukaj waluty po kodzie lub nazwie"
	currencyList.FilterInput.Prompt = "Szukaj: "
	currencyList.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("24"))

	cacheDelegate := list.NewDefaultDelegate()
	cacheDelegate.ShowDescription = false
	cacheDelegate.SetSpacing(0)
	cacheCurrencyList := list.New([]list.Item{}, cacheDelegate, 32, 12)
	cacheCurrencyList.Title = "PICKER WALUT"
	cacheCurrencyList.SetShowHelp(false)
	cacheCurrencyList.SetShowPagination(false)
	cacheCurrencyList.SetShowStatusBar(false)
	cacheCurrencyList.SetFilteringEnabled(true)
	cacheCurrencyList.SetShowFilter(true)
	cacheCurrencyList.DisableQuitKeybindings()
	cacheCurrencyList.FilterInput.Placeholder = "Filtruj waluty do pobrania"
	cacheCurrencyList.FilterInput.Prompt = "Filtr: "
	cacheCurrencyList.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("52"))

	dbDelegate := list.NewDefaultDelegate()
	dbDelegate.ShowDescription = true
	dbDelegate.SetSpacing(0)
	dbCurrencyList := list.New([]list.Item{}, dbDelegate, 32, 12)
	dbCurrencyList.Title = "WALUTY W BAZIE"
	dbCurrencyList.SetShowHelp(false)
	dbCurrencyList.SetShowPagination(true)
	dbCurrencyList.SetShowStatusBar(true)
	dbCurrencyList.SetFilteringEnabled(false)
	dbCurrencyList.DisableQuitKeybindings()
	dbCurrencyList.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("20"))

	plnAmountInput := textinput.New()
	plnAmountInput.Prompt = ""
	plnAmountInput.Placeholder = "np. 1500.00"
	plnAmountInput.CharLimit = 20
	plnAmountInput.SetValue("100.00")
	plnAmountInput.SetWidth(24)

	foreignAmountInput := textinput.New()
	foreignAmountInput.Prompt = ""
	foreignAmountInput.Placeholder = "np. 250.00"
	foreignAmountInput.CharLimit = 20
	foreignAmountInput.SetWidth(24)

	dateValue := cfg.LastConverterDate
	if dateValue == "" {
		dateValue = time.Now().Format("2006-01-02")
	}
	dateInput := textinput.New()
	dateInput.Prompt = ""
	dateInput.Placeholder = "YYYY-MM-DD"
	dateInput.SetValue(dateValue)
	dateInput.SetWidth(24)

	fromValue := cfg.LastFromDate
	if fromValue == "" {
		fromValue = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	cacheFromInput := textinput.New()
	cacheFromInput.Prompt = ""
	cacheFromInput.Placeholder = "YYYY-MM-DD"
	cacheFromInput.SetValue(fromValue)
	cacheFromInput.SetWidth(24)

	cacheToInput := textinput.New()
	cacheToInput.Prompt = ""
	cacheToInput.Placeholder = "YYYY-MM-DD"
	cacheToInput.SetValue(time.Now().Format("2006-01-02"))
	cacheToInput.SetWidth(24)

	dbFilterInput := textinput.New()
	dbFilterInput.Prompt = ""
	dbFilterInput.Placeholder = "filtr po kodzie lub nazwie"
	dbFilterInput.SetWidth(28)

	resultsViewport := viewport.New(viewport.WithWidth(60), viewport.WithHeight(14))
	resultsViewport.SetContent("WYBIERZ WALUTĘ Z PICKERA, WPISZ KWOTĘ I DATĘ, A NASTĘPNIE NACIŚNIJ [ PRZELICZ ].")

	cacheViewport := viewport.New(viewport.WithWidth(60), viewport.WithHeight(14))
	cacheViewport.SetContent("ŁADOWANIE INFORMACJI O BAZIE CACHE...")

	dbViewport := viewport.New(viewport.WithWidth(60), viewport.WithHeight(14))
	dbViewport.SetContent("ŁADOWANIE LISTY WALUT Z BAZY...")

	helpModel := help.New()
	helpModel.ShowAll = false

	spin := spinner.New(spinner.WithSpinner(spinner.Line))
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)

	prog := progress.New(progress.WithWidth(34))

	return tuiModel{
		configPath:              configPath,
		cfg:                     cfg,
		service:                 service,
		store:                   store,
		keys:                    defaultTUIKeyMap(),
		help:                    helpModel,
		spinner:                 spin,
		progress:                prog,
		currencyList:            currencyList,
		cacheCurrencyList:       cacheCurrencyList,
		dbCurrencyList:          dbCurrencyList,
		plnAmountInput:          plnAmountInput,
		foreignAmountInput:      foreignAmountInput,
		dateInput:               dateInput,
		cacheFromInput:          cacheFromInput,
		cacheToInput:            cacheToInput,
		dbFilterInput:           dbFilterInput,
		resultsViewport:         resultsViewport,
		cacheViewport:           cacheViewport,
		dbViewport:              dbViewport,
		status:                  "Gotowy",
		direction:               directionPLNToForeign,
		directionMode:           directionModeAuto,
		lastEditedAmount:        amountFieldPLN,
		dbSortMode:              dbSortByCode,
		selectedCacheCurrencies: map[string]bool{},
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(
		m.plnAmountInput.Focus(),
		textinput.Blink,
		loadCurrenciesCmd(m.service, m.cfg.TimeoutSeconds),
		loadCacheInfoCmd(m.store),
		loadCurrencyStatsCmd(m.store),
	)
}

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var backgroundCmds []tea.Cmd

	if m.loading || m.cacheBusy {
		var spinCmd tea.Cmd
		m.spinner, spinCmd = m.spinner.Update(msg)
		if spinCmd != nil {
			backgroundCmds = append(backgroundCmds, spinCmd)
		}
	}
	if m.cacheBusy {
		var progressCmd tea.Cmd
		m.progress, progressCmd = m.progress.Update(msg)
		if progressCmd != nil {
			backgroundCmds = append(backgroundCmds, progressCmd)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg, backgroundCmds)
	case currencyClickedMsg:
		return m.handleCurrencyClicked(msg, backgroundCmds)
	case mouseActionMsg:
		return m.handleMouseAction(msg, backgroundCmds)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case currenciesLoadedMsg:
		return m.handleCurrenciesLoaded(msg, backgroundCmds)
	case convertFinishedMsg:
		return m.handleConvertFinished(msg, backgroundCmds)
	case cacheInfoLoadedMsg:
		return m.handleCacheInfoLoaded(msg, backgroundCmds)
	case currencyStatsLoadedMsg:
		return m.handleCurrencyStatsLoaded(msg, backgroundCmds)
	case currencyHistoryLoadedMsg:
		return m.handleCurrencyHistoryLoaded(msg, backgroundCmds)
	case cacheClearedMsg:
		return m.handleCacheCleared(msg, backgroundCmds)
	case configSavedMsg:
		return m.handleConfigSaved(msg, backgroundCmds)
	case prefetchFinishedMsg:
		return m.handlePrefetchFinished(msg, backgroundCmds)
	case prefetchChunkFinishedMsg:
		return m.handlePrefetchChunkFinished(msg, backgroundCmds)
	}

	if m.activeTab == 0 {
		model, cmd := m.updateConverterComponents(msg)
		return model, batchCmds(append(backgroundCmds, cmd)...)
	}
	if m.activeTab == 2 {
		model, cmd := m.updateDatabaseComponents(msg)
		return model, batchCmds(append(backgroundCmds, cmd)...)
	}
	model, cmd := m.updateCacheComponents(msg)
	return model, batchCmds(append(backgroundCmds, cmd)...)
}

func (m *tuiModel) handleWindowSize(msg tea.WindowSizeMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.resize()
	m.refreshCacheViewport()
	m.refreshDBViewport()
	return m, batchCmds(backgroundCmds...)
}

func (m *tuiModel) handleCurrencyClicked(msg currencyClickedMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	if msg.target == "converter" {
		m.focus = 0
		m.currencyList.Select(msg.index)
		return m, batchCmds(backgroundCmds...)
	}
	if msg.target == "cache-picker" {
		m.activeTab = 1
		m.focus = 0
		m.cacheCurrencyList.Select(msg.index)
		return m, batchCmds(append(backgroundCmds, m.toggleCurrentCacheCurrency())...)
	}
	if msg.target == "db" {
		m.activeTab = 2
		m.focus = 2
		m.dbCurrencyList.Select(msg.index)
		return m.loadSelectedDBCurrencyHistory()
	}
	return m, batchCmds(backgroundCmds...)
}

func (m *tuiModel) handleMouseAction(msg mouseActionMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.action {
	case "convert":
		if !m.loading {
			return m.startConversion()
		}
		return m, batchCmds(backgroundCmds...)
	case "toggle-all-cache":
		m.activeTab = 1
		m.focus = 2
		m.allCacheCurrencies = !m.allCacheCurrencies
		m.status = m.cacheSelectionStatus()
		return m, batchCmds(backgroundCmds...)
	case "prefetch-cache":
		m.activeTab = 1
		m.focus = 2
		if !m.cacheBusy {
			return m.startRangePrefetch()
		}
		return m, batchCmds(backgroundCmds...)
	case "open-currency-picker":
		m.activeTab = 1
		m.focus = 3
		m.currencyPickerOpen = true
		m.currencyPickerTarget = pickerTargetCache
		return m, batchCmds(backgroundCmds...)
	case "close-currency-picker":
		m.currencyPickerOpen = false
		m.currencyPickerTarget = pickerTargetNone
		return m, batchCmds(append(backgroundCmds, m.focusCurrentField())...)
	case "focus-amount":
		m.activeTab = 0
		m.focus = 1
		m.lastEditedAmount = amountFieldPLN
		return m, batchCmds(append(backgroundCmds, m.focusCurrentField())...)
	case "focus-foreign-amount":
		m.activeTab = 0
		m.focus = 2
		m.lastEditedAmount = amountFieldForeign
		return m, batchCmds(append(backgroundCmds, m.focusCurrentField())...)
	case "focus-date":
		m.activeTab = 0
		m.focus = 3
		return m, batchCmds(append(backgroundCmds, m.focusCurrentField())...)
	case "focus-cache-from":
		m.activeTab = 1
		m.focus = 0
		return m, batchCmds(append(backgroundCmds, m.focusCurrentField())...)
	case "focus-cache-to":
		m.activeTab = 1
		m.focus = 1
		return m, batchCmds(append(backgroundCmds, m.focusCurrentField())...)
	case "focus-db-filter":
		m.activeTab = 2
		m.focus = 0
		return m, batchCmds(append(backgroundCmds, m.focusCurrentField())...)
	case "toggle-db-sort":
		m.activeTab = 2
		m.focus = 1
		m.dbSortMode = m.dbSortMode.next()
		m.rebuildDBCurrencyList()
		return m, batchCmds(backgroundCmds...)
	case "tab-converter":
		m.activeTab = 0
		m.focus = 0
		return m, batchCmds(append(backgroundCmds, m.focusCurrentField())...)
	case "tab-cache":
		m.activeTab = 1
		m.focus = 0
		return m, batchCmds(append(backgroundCmds, loadCacheInfoCmd(m.store))...)
	case "tab-db":
		m.activeTab = 2
		m.focus = 0
		return m, batchCmds(append(backgroundCmds, loadCurrencyStatsCmd(m.store))...)
	}
	return m, batchCmds(backgroundCmds...)
}

func (m *tuiModel) handleCurrenciesLoaded(msg currenciesLoadedMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.lastError = humanizeError(msg.err)
		m.status = "Nie udało się załadować listy walut"
		return m, batchCmds(backgroundCmds...)
	}
	m.lastError = ""
	m.currenciesReady = true
	m.currencies = prioritizeCurrencies(append([]models.Currency(nil), msg.currencies...))
	items := make([]list.Item, 0, len(m.currencies))
	selectedIndex := 0
	for i, currency := range m.currencies {
		items = append(items, converterCurrencyItem{code: currency.Code, name: currency.Name})
		if strings.EqualFold(currency.Code, "USD") {
			selectedIndex = i
		}
	}
	cmd := m.currencyList.SetItems(items)
	m.currencyList.Select(selectedIndex)
	m.status = fmt.Sprintf("Załadowano %d walut NBP", len(msg.currencies))
	return m, batchCmds(append(backgroundCmds, cmd)...)
}

func prioritizeCurrencies(currencies []models.Currency) []models.Currency {
	preferredOrder := map[string]int{
		"USD": 0,
		"EUR": 1,
		"CHF": 2,
		"GBP": 3,
	}

	sort.SliceStable(currencies, func(i, j int) bool {
		leftRank, leftPreferred := preferredOrder[strings.ToUpper(currencies[i].Code)]
		rightRank, rightPreferred := preferredOrder[strings.ToUpper(currencies[j].Code)]

		if leftPreferred && rightPreferred {
			return leftRank < rightRank
		}
		if leftPreferred != rightPreferred {
			return leftPreferred
		}
		return false
	})

	return currencies
}

func (m *tuiModel) handleConvertFinished(msg convertFinishedMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.lastError = humanizeError(msg.err)
		m.status = "Błąd przeliczenia"
		return m, batchCmds(backgroundCmds...)
	}
	m.lastError = ""
	m.direction = msg.direction
	if msg.direction == directionPLNToForeign {
		m.plnAmountInput.SetValue(formatAmount(msg.sourceAmount))
		m.foreignAmountInput.SetValue(formatAmount(msg.targetAmount))
	} else {
		m.foreignAmountInput.SetValue(formatAmount(msg.sourceAmount))
		m.plnAmountInput.SetValue(formatAmount(msg.targetAmount))
	}
	m.status = "Przeliczenie zakończone"
	m.resultsViewport.SetContent(msg.result)
	m.resultsViewport.GotoTop()
	return m, batchCmds(append(backgroundCmds, loadCacheInfoCmd(m.store))...)
}

func (m *tuiModel) handleCacheInfoLoaded(msg cacheInfoLoadedMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	m.cacheBusy = false
	if msg.err != nil {
		m.lastError = humanizeError(msg.err)
		m.status = "Błąd odczytu bazy cache"
		return m, batchCmds(backgroundCmds...)
	}
	m.lastError = ""
	m.cacheInfo = msg.info
	m.refreshCacheViewport()
	if m.activeTab == 1 {
		m.status = "Odczytano statystyki bazy walut"
	}
	return m, batchCmds(backgroundCmds...)
}

func (m *tuiModel) handleCurrencyStatsLoaded(msg currencyStatsLoadedMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.lastError = humanizeError(msg.err)
		if m.activeTab == 2 {
			m.status = "Błąd odczytu walut z bazy"
		}
		return m, batchCmds(backgroundCmds...)
	}
	m.lastError = ""
	m.currencyStats = append([]cache.CurrencyStat(nil), msg.stats...)
	m.rebuildDBCurrencyList()
	if _, ok := m.selectedDBStat(); ok && len(m.selectedDBHistory) == 0 {
		model, cmd := m.loadSelectedDBCurrencyHistory()
		return model, batchCmds(append(backgroundCmds, cmd)...)
	}
	if m.activeTab == 2 {
		m.status = fmt.Sprintf("Odczytano %d walut z bazy", len(msg.stats))
	}
	return m, batchCmds(backgroundCmds...)
}

func (m *tuiModel) handleCurrencyHistoryLoaded(msg currencyHistoryLoadedMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.lastError = humanizeError(msg.err)
		m.status = "Błąd odczytu historii waluty"
		return m, batchCmds(backgroundCmds...)
	}
	m.lastError = ""
	m.selectedDBCurrency = msg.currency
	m.selectedDBHistory = append([]cache.CurrencyHistoryEntry(nil), msg.history...)
	m.refreshDBViewport()
	m.status = fmt.Sprintf("Odczytano historię waluty %s", msg.currency)
	return m, batchCmds(backgroundCmds...)
}

func (m *tuiModel) handleCacheCleared(msg cacheClearedMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	m.cacheBusy = false
	if msg.err != nil {
		m.lastError = humanizeError(msg.err)
		m.status = "Błąd czyszczenia bazy cache"
		return m, batchCmds(backgroundCmds...)
	}
	m.lastError = ""
	m.status = "Baza cache wyczyszczona"
	m.resultsViewport.SetContent("CACHE ZOSTAŁ WYCZYSZCZONY. NOWE KURSY BĘDĄ DOPISYWANE PRZY KOLEJNYCH ZAPYTANIACH.")
	return m, batchCmds(append(backgroundCmds, loadCacheInfoCmd(m.store), loadCurrencyStatsCmd(m.store), loadCurrenciesCmd(m.service, m.cfg.TimeoutSeconds))...)
}

func (m *tuiModel) handleConfigSaved(msg configSavedMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.lastError = humanizeError(msg.err)
		m.status = "Nie udało się zapisać ustawień"
		return m, batchCmds(backgroundCmds...)
	}
	return m, batchCmds(backgroundCmds...)
}

func (m *tuiModel) handlePrefetchFinished(msg prefetchFinishedMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	m.cacheBusy = false
	if msg.err != nil {
		m.lastError = humanizeError(msg.err)
		m.status = "Błąd pobierania zakresu do bazy walut"
		return m, batchCmds(backgroundCmds...)
	}
	m.lastError = ""
	m.prefetchChunks = nil
	m.prefetchSummary = msg.summary
	m.status = fmt.Sprintf("Baza walut uzupełniona: %d kursów dla %d walut", msg.summary.RateCount, msg.summary.CurrencyCount)
	m.refreshCacheViewport()
	return m, batchCmds(append(backgroundCmds, loadCacheInfoCmd(m.store), loadCurrencyStatsCmd(m.store))...)
}

func (m *tuiModel) handlePrefetchChunkFinished(msg prefetchChunkFinishedMsg, backgroundCmds []tea.Cmd) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.cacheBusy = false
		m.prefetchChunks = nil
		m.lastError = humanizeError(msg.err)
		m.status = "Błąd pobierania zakresu do bazy walut"
		return m, batchCmds(backgroundCmds...)
	}

	m.prefetchDone++
	m.prefetchSummary.RateCount += msg.rateCount

	cmds := append([]tea.Cmd{}, backgroundCmds...)
	total := len(m.prefetchChunks) + m.prefetchDone
	if total > 0 {
		cmds = append(cmds, m.progress.SetPercent(float64(m.prefetchDone)/float64(total)))
	}

	if len(m.prefetchChunks) == 0 {
		m.cacheBusy = false
		m.status = fmt.Sprintf("Baza walut uzupełniona: %d kursów dla %d walut", m.prefetchSummary.RateCount, m.prefetchSummary.CurrencyCount)
		cmds = append(cmds, tea.Cmd(func() tea.Msg {
			return prefetchFinishedMsg{summary: m.prefetchSummary}
		}))
		return m, tea.Batch(cmds...)
	}

	next := m.prefetchChunks[0]
	m.prefetchChunks = m.prefetchChunks[1:]
	m.status = fmt.Sprintf("Import do cache: %s %s -> %s", next.currency, next.start.Format("2006-01-02"), next.end.Format("2006-01-02"))
	cmds = append(cmds, prefetchChunkCmd(m.service, m.cfg.TimeoutSeconds, next), spinnerTickCmd(m.spinner))
	return m, tea.Batch(cmds...)
}

func (m tuiModel) View() tea.View {
	// W Bubble Tea v2 Width i Height na stylu to CAŁKOWITY rozmiar bloku.
	// Ustawiamy ramkę na dokładny wymiar terminala.
	shell := shellStyle(m.width, m.height)

	// Zawartość wewnątrz ramki ma szerokość i wysokość o 4 mniejszą (2 obramowanie + 2 padding).
	innerWidth := m.width - 4
	innerHeight := m.height - 4
	if innerWidth < 10 {
		innerWidth = 10
	}
	if innerHeight < 10 {
		innerHeight = 10
	}

	header := m.renderHeader(innerWidth)
	status := m.renderStatus(innerWidth)
	footer := m.renderFooter(innerWidth)

	headerHeight := lipgloss.Height(header)
	statusHeight := lipgloss.Height(status)
	footerHeight := lipgloss.Height(footer)

	remainingHeight := innerHeight - headerHeight - statusHeight - footerHeight
	if remainingHeight < 0 {
		remainingHeight = 0
	}

	bodyWidth := maxInt(innerWidth-2, 10)
	bodyHeight := maxInt(remainingHeight-2, 3)
	body := m.renderBody(bodyWidth, bodyHeight)
	expandedBody := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(innerWidth).
		Height(remainingHeight).
		Render(body)

	content := lipgloss.JoinVertical(lipgloss.Left, header, expandedBody, status, footer)

	v := tea.NewView(shell.Render(content))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.OnMouse = m.onMouse()
	return v
}

func (m *tuiModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.currencyPickerOpen {
		return m.handleCurrencyPickerKey(msg)
	}
	if m.activeTab == 0 && m.focus == 0 && m.currencyList.SettingFilter() {
		return m.handleConverterKey(msg)
	}
	if m.activeTab == 1 && m.cacheCurrencyList.SettingFilter() {
		return m.handleCacheKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case msg.String() == "esc":
		if m.textInputFocused() {
			m.focus = 0
			return m, m.focusCurrentField()
		}
		return m, nil
	case m.canSwitchToPrevTab(msg):
		return m.switchTab(-1)
	case m.canSwitchToNextTab(msg):
		return m.switchTab(1)
	case key.Matches(msg, m.keys.ToggleHelp):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return m, nil
	}

	if m.activeTab == 0 {
		return m.handleConverterKey(msg)
	}
	if m.activeTab == 1 {
		return m.handleCacheKey(msg)
	}
	return m.handleDBKey(msg)
}

func (m *tuiModel) switchTab(delta int) (tea.Model, tea.Cmd) {
	m.activeTab = (m.activeTab + delta + 3) % 3
	m.focus = 0

	switch m.activeTab {
	case 0:
		return m, m.focusCurrentField()
	case 1:
		return m, batchCmds(m.focusCurrentField(), loadCacheInfoCmd(m.store))
	default:
		return m, loadCurrencyStatsCmd(m.store)
	}
}

func (m *tuiModel) handleDBKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focus = (m.focus + 1) % 4
		return m, m.focusCurrentField()
	case "shift+tab":
		m.focus = (m.focus + 3) % 4
		return m, m.focusCurrentField()
	case "down":
		if m.focus < 3 && m.focus != 2 {
			m.focus++
			return m, m.focusCurrentField()
		}
	case "up":
		if m.focus > 0 && m.focus != 2 {
			m.focus--
			return m, m.focusCurrentField()
		}
	case "enter":
		if m.focus == 1 {
			m.dbSortMode = m.dbSortMode.next()
			m.rebuildDBCurrencyList()
			return m, nil
		}
		if m.focus == 2 {
			return m.loadSelectedDBCurrencyHistory()
		}
	case "/":
		m.focus = 0
		return m, m.focusCurrentField()
	}

	if msg.String() == "r" {
		m.status = "Odświeżanie widoku walut w bazie..."
		return m, loadCurrencyStatsCmd(m.store)
	}

	return m.updateDatabaseComponents(msg)
}

func (m *tuiModel) handleConverterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focus = (m.focus + 1) % 5
		return m, m.focusCurrentField()
	case "shift+tab":
		m.focus = (m.focus + 4) % 5
		return m, m.focusCurrentField()
	case "down":
		if m.focus > 0 && m.focus < 4 {
			m.focus++
			return m, m.focusCurrentField()
		}
	case "up":
		if m.focus > 0 {
			m.focus--
			return m, m.focusCurrentField()
		}
	case "enter":
		if m.focus == 4 && !m.loading {
			return m.startConversion()
		}
		if m.focus == 1 || m.focus == 2 || m.focus == 3 {
			m.focus = (m.focus + 1) % 5
			return m, m.focusCurrentField()
		}
	}
	if key.Matches(msg, m.keys.ToggleDir) {
		m.directionMode = directionModeManual
		m.direction = m.converterDirection().next()
		m.lastError = ""
		m.status = "Ręcznie przełączono kierunek konwersji"
		return m, nil
	}

	return m.updateConverterComponents(msg)
}

func (m *tuiModel) handleCacheKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focus = (m.focus + 1) % 3
		return m, m.focusCurrentField()
	case "shift+tab":
		m.focus = (m.focus + 2) % 3
		return m, m.focusCurrentField()
	case "down":
		if m.focus < 2 {
			m.focus++
			return m, m.focusCurrentField()
		}
	case "up":
		if m.focus > 0 {
			m.focus--
			return m, m.focusCurrentField()
		}
	case "enter":
		if m.focus == 2 && !m.cacheBusy {
			return m.startRangePrefetch()
		}
	case "r":
		if m.cacheBusy {
			return m, nil
		}
		m.cacheBusy = true
		m.status = "Odświeżanie statystyk bazy cache..."
		return m, tea.Batch(loadCacheInfoCmd(m.store), spinnerTickCmd(m.spinner))
	case "c":
		if m.cacheBusy {
			return m, nil
		}
		m.cacheBusy = true
		m.status = "Czyszczenie bazy cache..."
		return m, tea.Batch(clearCacheCmd(m.store), spinnerTickCmd(m.spinner))
	}
	return m.updateCacheComponents(msg)
}

func (m *tuiModel) handleCurrencyPickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case msg.String() == "esc":
		m.currencyPickerOpen = false
		m.currencyPickerTarget = pickerTargetNone
		return m, m.focusCurrentField()
	}

	if m.cacheCurrencyList.SettingFilter() {
		if msg.String() == "enter" {
			return m, m.toggleCurrentCacheCurrency()
		}
		var cmd tea.Cmd
		m.cacheCurrencyList, cmd = m.cacheCurrencyList.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "enter", " ":
		return m, m.toggleCurrentCacheCurrency()
	}

	if key.Matches(msg, m.keys.ToggleAll) {
		m.allCacheCurrencies = !m.allCacheCurrencies
		m.status = m.cacheSelectionStatus()
		return m, m.syncCacheCurrencyList()
	}

	var cmd tea.Cmd
	m.cacheCurrencyList, cmd = m.cacheCurrencyList.Update(msg)
	return m, cmd
}

func (m *tuiModel) updateConverterComponents(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if m.focus == 0 {
		acceptFilteredSelection := false
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			acceptFilteredSelection = keyMsg.String() == "enter" && m.currencyList.SettingFilter()
		}
		m.currencyList, cmd = m.currencyList.Update(msg)
		if acceptFilteredSelection {
			m.resetCurrencyFilterToSelection()
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.focus == 1 {
		before := m.plnAmountInput.Value()
		m.plnAmountInput, cmd = m.plnAmountInput.Update(msg)
		if m.plnAmountInput.Value() != before {
			m.lastEditedAmount = amountFieldPLN
			m.directionMode = directionModeAuto
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.focus == 2 {
		before := m.foreignAmountInput.Value()
		m.foreignAmountInput, cmd = m.foreignAmountInput.Update(msg)
		if m.foreignAmountInput.Value() != before {
			m.lastEditedAmount = amountFieldForeign
			m.directionMode = directionModeAuto
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.focus == 3 {
		m.dateInput, cmd = m.dateInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	m.resultsViewport, cmd = m.resultsViewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *tuiModel) resetCurrencyFilterToSelection() {
	selected, ok := m.selectedCurrency()
	m.currencyList.ResetFilter()
	if !ok {
		return
	}

	for i, currency := range m.currencies {
		if strings.EqualFold(currency.Code, selected.Code) {
			m.currencyList.Select(i)
			return
		}
	}
}

func (m *tuiModel) updateCacheComponents(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if m.currencyPickerOpen {
		m.cacheCurrencyList, cmd = m.cacheCurrencyList.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	if m.focus == 0 {
		m.cacheFromInput, cmd = m.cacheFromInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.focus == 1 {
		m.cacheToInput, cmd = m.cacheToInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	m.cacheViewport, cmd = m.cacheViewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *tuiModel) updateDatabaseComponents(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if m.focus == 0 {
		before := m.dbFilterInput.Value()
		m.dbFilterInput, cmd = m.dbFilterInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if before != m.dbFilterInput.Value() {
			m.rebuildDBCurrencyList()
		}
	}
	if m.focus == 2 {
		beforeCode := m.selectedDBCurrency
		m.dbCurrencyList, cmd = m.dbCurrencyList.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if stat, ok := m.selectedDBStat(); ok && !strings.EqualFold(stat.Code, beforeCode) {
			m.selectedDBCurrency = stat.Code
			m.selectedDBHistory = nil
			cmds = append(cmds, loadCurrencyHistoryCmd(m.store, stat.Code, 120))
		}
	}

	m.dbViewport, cmd = m.dbViewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *tuiModel) startConversion() (tea.Model, tea.Cmd) {
	if !m.currenciesReady {
		m.lastError = "lista walut nie jest jeszcze gotowa"
		return m, nil
	}
	currency, ok := m.selectedCurrency()
	if !ok {
		m.lastError = "nie wybrano waluty"
		return m, nil
	}

	m.cfg.LastConverterDate = m.dateInput.Value()

	m.loading = true
	m.lastError = ""
	m.status = "Pobieranie kursu i przeliczanie kwoty..."
	return m, tea.Batch(
		convertCurrencyCmd(m.service, m.cfg.TimeoutSeconds, currency, m.converterSourceAmount(), m.dateInput.Value(), m.converterDirection()),
		saveConfigCmd(m.configPath, m.cfg),
		spinnerTickCmd(m.spinner),
	)
}

func (m *tuiModel) focusCurrentField() tea.Cmd {
	m.plnAmountInput.Blur()
	m.foreignAmountInput.Blur()
	m.dateInput.Blur()
	m.cacheFromInput.Blur()
	m.cacheToInput.Blur()
	m.dbFilterInput.Blur()

	if m.activeTab == 0 {
		switch m.focus {
		case 1:
			return m.plnAmountInput.Focus()
		case 2:
			return m.foreignAmountInput.Focus()
		case 3:
			return m.dateInput.Focus()
		}
		return nil
	}
	if m.activeTab == 2 {
		if m.focus == 0 {
			return m.dbFilterInput.Focus()
		}
		return nil
	}

	switch m.focus {
	case 0:
		return m.cacheFromInput.Focus()
	case 1:
		return m.cacheToInput.Focus()
	default:
		return nil
	}
}

func (m *tuiModel) resize() {
	bodyWidth := maxInt(m.shellInnerWidth()-2, 40)
	bodyHeight := maxInt(m.bodyAreaHeight(), 10)

	converterListWidth, converterResultWidth, converterStacked := splitPaneWidths(bodyWidth, 28, 32)
	if converterStacked {
		converterListWidth = bodyWidth
		converterResultWidth = bodyWidth
	}
	m.currencyList.SetSize(maxInt(converterListWidth-4, 18), maxInt(bodyHeight-4, 6))
	m.plnAmountInput.SetWidth(maxInt(converterResultWidth-8, 18))
	m.foreignAmountInput.SetWidth(maxInt(converterResultWidth-8, 18))
	m.dateInput.SetWidth(maxInt(converterResultWidth-8, 18))
	m.resultsViewport.SetWidth(maxInt(converterResultWidth-6, 18))
	m.resultsViewport.SetHeight(maxInt(bodyHeight/2-4, 6))

	cacheFormWidth, cacheStatsWidth, cacheStacked := splitPaneWidths(bodyWidth, 34, 32)
	if cacheStacked {
		cacheFormWidth = bodyWidth
		cacheStatsWidth = bodyWidth
	}
	m.cacheCurrencyList.SetSize(maxInt(minInt(bodyWidth-10, 36), 18), minInt(maxInt(bodyHeight-12, 6), 10))
	m.cacheFromInput.SetWidth(maxInt(cacheFormWidth-8, 18))
	m.cacheToInput.SetWidth(maxInt(cacheFormWidth-8, 18))
	m.cacheViewport.SetWidth(maxInt(cacheStatsWidth-6, 18))
	m.cacheViewport.SetHeight(maxInt(bodyHeight-6, 6))

	dbListWidth, dbDetailWidth, dbStacked := splitPaneWidths(bodyWidth, 28, 32)
	if dbStacked {
		dbListWidth = bodyWidth
		dbDetailWidth = bodyWidth
	}
	m.dbFilterInput.SetWidth(maxInt(dbListWidth-6, 18))
	m.dbCurrencyList.SetSize(maxInt(dbListWidth-4, 18), maxInt(bodyHeight-12, 4))
	m.dbViewport.SetWidth(maxInt(dbDetailWidth-6, 18))
	m.dbViewport.SetHeight(maxInt(bodyHeight-5, 4))
	m.help.SetWidth(maxInt(m.shellInnerWidth(), 24))
}

func (m tuiModel) renderHeader(innerWidth int) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderTitleBar(innerWidth),
		m.renderHeaderDivider(innerWidth),
		m.renderTabs(),
	)
}

func (m tuiModel) renderTitleBar(innerWidth int) string {
	title := "KURSOMAT"
	subtitle := "Terminalowy interfejs kursów walut z cache SQLite"
	if m.activeTab == 1 {
		subtitle = "Import i zarządzanie bazą cache"
	} else if m.activeTab == 2 {
		subtitle = "Przegląd walut zapisanych w bazie"
	}

	return lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("24")).
		Width(innerWidth).
		Render(title + " • " + subtitle)
}

func (m tuiModel) renderHeaderDivider(innerWidth int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Width(innerWidth).
		Render(strings.Repeat("─", maxInt(innerWidth, 1)))
}

func (m tuiModel) renderTabs() string {
	tabBase := lipgloss.NewStyle().
		Padding(0, 2).
		Bold(true)
	active := tabBase.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("25"))
	inactive := tabBase.Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238"))

	converterTab := inactive.Render("KONWERTER")
	walletTab := inactive.Render("BAZA WALUT")
	dbTab := inactive.Render("WALUTY W BAZIE")
	if m.activeTab == 0 {
		converterTab = active.Render("KONWERTER")
	} else if m.activeTab == 1 {
		walletTab = active.Render("BAZA WALUT")
	} else {
		dbTab = active.Render("WALUTY W BAZIE")
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, converterTab, " ", walletTab, " ", dbTab)
}

func (m tuiModel) renderBody(innerWidth, innerHeight int) string {
	if m.currencyPickerOpen && m.currencyPickerTarget == pickerTargetCache {
		return lipgloss.Place(
			innerWidth,
			innerHeight,
			lipgloss.Center,
			lipgloss.Center,
			m.renderCurrencyPickerModal(innerWidth, innerHeight),
		)
	}
	if m.activeTab == 0 {
		return lipgloss.Place(innerWidth, innerHeight, lipgloss.Left, lipgloss.Top, m.renderConverterBody(innerWidth, innerHeight))
	}
	if m.activeTab == 2 {
		return lipgloss.Place(innerWidth, innerHeight, lipgloss.Left, lipgloss.Top, m.renderDatabaseBody(innerWidth, innerHeight))
	}
	return lipgloss.Place(innerWidth, innerHeight, lipgloss.Left, lipgloss.Top, m.renderCacheBody(innerWidth))
}

func (m tuiModel) renderConverterBody(innerWidth, innerHeight int) string {
	listWidth, rightWidth, stacked := splitPaneWidths(innerWidth, 28, 32)
	if stacked {
		listWidth = innerWidth
		rightWidth = innerWidth
	}
	listCardStyle := cardStyle(m.focus == 0).Width(listWidth)
	rightCardStyle := cardStyle(m.focus >= 1).Width(rightWidth)
	if !stacked {
		listCardStyle = listCardStyle.Height(innerHeight)
		rightCardStyle = rightCardStyle.Height(innerHeight)
	}
	listCard := listCardStyle.Render(m.currencyList.View())

	selectedCode := "?"
	selectedName := "brak wybranej waluty"
	if currency, ok := m.selectedCurrency(); ok {
		selectedCode = currency.Code
		selectedName = currency.Name
	}

	convertButtonLabel := "[ PRZELICZ ]"
	if m.loading {
		convertButtonLabel = m.spinner.View() + " PRZELICZANIE"
	}
	convertButton := primaryButtonStyle(m.focus == 4 && !m.loading).Render(convertButtonLabel)
	autoDirection := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248")).
		Render(m.directionStatusLabel(selectedCode))

	resultTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")).
		Background(lipgloss.Color("18")).
		Padding(0, 1).
		Render(strings.ToUpper("wynik i kurs z dnia"))

	activeCurrencyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")).
		Background(lipgloss.Color("238")).
		Padding(0, 1).
		MarginBottom(1)

	formLines := []string{
		sectionTitle("WALUTA"),
		activeCurrencyStyle.Render(fmt.Sprintf("%s - %s", strings.ToUpper(selectedCode), selectedName)),
		"",
		sectionTitle("KWOTA PLN"),
		m.plnAmountInput.View(),
		"",
		sectionTitle("KWOTA " + strings.ToUpper(selectedCode)),
		m.foreignAmountInput.View(),
		"",
		autoDirection,
		"",
		sectionTitle("DZIEŃ KURSU (RRRR-MM-DD)"),
		m.dateInput.View(),
		"",
		convertButton,
	}
	rightCard := rightCardStyle.Render(strings.Join(append(formLines, "", resultTitle, m.resultsViewport.View()), "\n"))

	if stacked {
		return lipgloss.JoinVertical(lipgloss.Left, listCard, rightCard)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, listCard, " ", rightCard)
}

func (m tuiModel) renderCacheBody(innerWidth int) string {
	formWidth, statsWidth, stacked := splitPaneWidths(innerWidth, 34, 32)
	if stacked {
		formWidth = innerWidth
		statsWidth = innerWidth
	}
	importLabel := "[ POBIERZ ZAKRES ]"
	if m.cacheBusy {
		importLabel = m.spinner.View() + " POBIERANIE"
	}
	importButton := primaryButtonStyle(m.focus == 2 && !m.cacheBusy).Render(importLabel)

	formCard := cardStyle(m.focus <= 3).Width(formWidth).Render(strings.Join([]string{
		sectionTitle("ZAKRES DAT DO BAZY"),
		"Od:",
		m.cacheFromInput.View(),
		"",
		"Do:",
		m.cacheToInput.View(),
		"",
		sectionTitle("ZAKRES IMPORTU"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Render("Import obejmuje wszystkie waluty z tabeli A."),
		lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render("Ustawiasz tylko datę początkową i końcową."),
		"",
		sectionTitle("AKCJA"),
		importButton,
		"",
		sectionTitle("POSTĘP IMPORTU"),
		m.progress.ViewAs(m.cachePercent()),
	}, "\n"))

	statsBody := m.cacheViewport.View()
	if m.cacheBusy {
		statsBody = m.spinner.View() + " AKTUALIZACJA BAZY WALUT\n\n" + statsBody
	}
	statsCard := cardStyle(false).Width(statsWidth).Render(statsBody)

	if stacked {
		return lipgloss.JoinVertical(lipgloss.Left, formCard, " ", statsCard)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, formCard, " ", statsCard)
}

func (m tuiModel) renderDatabaseBody(innerWidth, innerHeight int) string {
	listWidth, detailWidth, stacked := splitPaneWidths(innerWidth, 28, 32)
	if stacked {
		listWidth = innerWidth
		detailWidth = innerWidth
	}
	filterCardStyle := cardStyle(m.focus == 0 || m.focus == 1 || m.focus == 2).Width(listWidth)
	detailCardStyle := cardStyle(m.focus == 3).Width(detailWidth)
	if !stacked {
		filterCardStyle = filterCardStyle.Height(innerHeight).MaxHeight(innerHeight)
		detailCardStyle = detailCardStyle.Height(innerHeight).MaxHeight(innerHeight)
	}

	filterCard := filterCardStyle.Render(strings.Join([]string{
		sectionTitle("FILTR"),
		m.dbFilterInput.View(),
		"",
		sectionTitle("SORTOWANIE"),
		secondaryButtonStyle(m.focus == 1).Render(m.dbSortMode.label()),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Render(fmt.Sprintf("Walut po filtrze: %d", len(m.filteredCurrencyStats))),
		"",
		m.dbCurrencyList.View(),
	}, "\n"))

	detailCard := detailCardStyle.Render(strings.Join([]string{
		sectionTitle("SZCZEGÓŁY WALUTY"),
		m.dbViewport.View(),
	}, "\n"))

	if stacked {
		return lipgloss.JoinVertical(lipgloss.Left, filterCard, " ", detailCard)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, filterCard, " ", detailCard)
}

func (m tuiModel) renderCurrencyPickerModal(innerWidth, innerHeight int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("24")).
		Padding(0, 2)

	availableWidth := maxInt(innerWidth-4, 24)
	availableHeight := maxInt(innerHeight-2, 8)
	modalWidth := minInt(availableWidth, 64)
	if modalWidth < 32 {
		modalWidth = availableWidth
	}
	modalHeight := minInt(availableHeight, 22)
	if modalHeight < 12 {
		modalHeight = availableHeight
	}
	modalInnerWidth := maxInt(modalWidth-6, 18)
	modalListHeight := maxInt(modalHeight-10, 3)

	allToggleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")).
		Padding(0, 1)

	if m.allCacheCurrencies {
		allToggleStyle = allToggleStyle.Foreground(lipgloss.Color("10")).Background(lipgloss.Color("22"))
	}

	allLabel := "[ ] WSZYSTKIE WALUTY (pobierze wszystko z tabeli A)"
	if m.allCacheCurrencies {
		allLabel = "[x] WSZYSTKIE WALUTY (wybrane wszystkie)"
	}

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248")).
		PaddingTop(1).
		Render("Enter/Spacja: zaznacz • a: przełącz wszystko • Esc: zamknij")

	m.cacheCurrencyList.SetSize(modalInnerWidth, modalListHeight)
	content := strings.Join([]string{
		titleStyle.Render("WYBÓR WALUT DO IMPORTU"),
		"",
		allToggleStyle.Render(allLabel),
		"",
		m.cacheCurrencyList.View(),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("248")).Render(m.cacheSelectionStatus()),
		lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(m.cacheSelectionPreview()),
		footer,
	}, "\n")

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("14")).
		Padding(1, 2).
		MaxWidth(modalWidth).
		MaxHeight(modalHeight).
		Width(modalWidth).
		Height(modalHeight).
		Render(content)
}

func (m tuiModel) renderStatus(innerWidth int) string {
	style := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("238")).
		Width(innerWidth).
		Height(1)

	if m.lastError != "" {
		return style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("1")).Render("BŁĄD: " + strings.ToUpper(m.lastError))
	}
	return style.Render("STATUS: " + strings.ToUpper(m.status))
}

func (m tuiModel) renderFooter(innerWidth int) string {
	style := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("250")).
		Background(lipgloss.Color("236")).
		Width(innerWidth)
	if m.activeTab == 1 {
		return style.Render(m.help.View(cacheHelpMap{keys: m.keys}))
	}
	if m.activeTab == 2 {
		return style.Render(m.help.View(databaseHelpMap{keys: m.keys}))
	}
	return style.Render(m.help.View(converterHelpMap{keys: m.keys}))
}

func (m tuiModel) selectedCurrency() (models.Currency, bool) {
	item, ok := m.currencyList.SelectedItem().(converterCurrencyItem)
	if !ok {
		return models.Currency{}, false
	}
	return models.Currency{Code: item.code, Name: item.name}, true
}

func (m tuiModel) converterDirection() conversionDirection {
	if m.directionMode == directionModeManual {
		return m.direction
	}
	if m.lastEditedAmount == amountFieldForeign {
		return directionForeignToPLN
	}
	return directionPLNToForeign
}

func (m tuiModel) directionStatusLabel(code string) string {
	if m.directionMode == directionModeManual {
		return "Tryb kierunku: RĘCZNY - " + m.converterDirection().label(code)
	}
	return "Tryb kierunku: AUTO - " + m.converterDirection().label(code)
}

func (m tuiModel) converterSourceAmount() string {
	if m.converterDirection() == directionForeignToPLN {
		return m.foreignAmountInput.Value()
	}
	return m.plnAmountInput.Value()
}

func (m tuiModel) converterInputFocused() bool {
	return m.activeTab == 0 && (m.focus == 1 || m.focus == 2 || m.focus == 3)
}

func (m tuiModel) textInputFocused() bool {
	return (m.activeTab == 0 && (m.focus == 1 || m.focus == 2 || m.focus == 3)) ||
		(m.activeTab == 1 && (m.focus == 0 || m.focus == 1)) ||
		(m.activeTab == 2 && m.focus == 0)
}

func (m tuiModel) canSwitchToPrevTab(msg tea.KeyPressMsg) bool {
	return msg.String() == "ctrl+shift+tab" || (key.Matches(msg, m.keys.PrevTab) && !m.textInputFocused())
}

func (m tuiModel) canSwitchToNextTab(msg tea.KeyPressMsg) bool {
	return msg.String() == "ctrl+tab" || (key.Matches(msg, m.keys.NextTab) && !m.textInputFocused())
}

func (m *tuiModel) onMouse() func(msg tea.MouseMsg) tea.Cmd {
	// Zawartość ramki zaczyna się po obramowaniu (1) i paddingu (1) = 2.
	contentX := 2
	contentY := 2

	return func(msg tea.MouseMsg) tea.Cmd {
		mouse := msg.Mouse()
		if mouse.Button != tea.MouseLeft {
			return nil
		}

		headerTopHeight := lipgloss.Height(lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderTitleBar(maxInt(m.width-4, 10)),
			m.renderHeaderDivider(maxInt(m.width-4, 10)),
		))
		tabY := contentY + headerTopHeight
		if hitRect(mouse.X, mouse.Y, contentX, tabY, 14, 1) {
			return func() tea.Msg { return mouseActionMsg{action: "tab-converter"} }
		}
		if hitRect(mouse.X, mouse.Y, contentX+15, tabY, 16, 1) {
			return func() tea.Msg { return mouseActionMsg{action: "tab-cache"} }
		}
		if hitRect(mouse.X, mouse.Y, contentX+32, tabY, 20, 1) {
			return func() tea.Msg { return mouseActionMsg{action: "tab-db"} }
		}

		bodyX := contentX + 1
		bodyY := contentY + lipgloss.Height(m.renderHeader(maxInt(m.width-4, 10))) + 1
		bodyWidth := maxInt(m.shellInnerWidth()-2, 40)
		bodyHeight := maxInt(m.bodyAreaHeight(), 10)

		if m.currencyPickerOpen {
			innerWidth := m.width - 4
			innerHeight := m.height - 4
			modal := m.renderCurrencyPickerModal(innerWidth, innerHeight)
			modalWidth := lipgloss.Width(modal)
			modalHeight := lipgloss.Height(modal)

			// Obliczamy pozycję modala (jest wycentrowany w bodyArea)
			modalX := bodyX + (maxInt(innerWidth-2, 10)-modalWidth)/2
			modalY := bodyY + (m.bodyAreaHeight()-modalHeight)/2

			listX := modalX + 2
			listY := modalY + 6
			target := "cache-picker"
			model := m.cacheCurrencyList
			if hitRect(mouse.X, mouse.Y, listX, listY, modalWidth-4, modalHeight-10) {
				row := mouse.Y - listY
				pageStart := model.Index() - model.Cursor()
				itemIndex := pageStart + row
				visibleItems := model.VisibleItems()
				if itemIndex >= 0 && itemIndex < len(visibleItems) {
					return func() tea.Msg {
						return currencyClickedMsg{target: target, index: itemIndex}
					}
				}
			}
			return func() tea.Msg { return mouseActionMsg{action: "close-currency-picker"} }
		}

		if m.activeTab == 0 {
			listWidth, rightWidth, stacked := splitPaneWidths(bodyWidth, 28, 32)
			listX := bodyX
			listY := bodyY
			listCardStyle := cardStyle(m.focus == 0).Width(listWidth)
			if !stacked {
				listCardStyle = listCardStyle.Height(bodyHeight)
			}
			listCardRendered := listCardStyle.Render(m.currencyList.View())
			listCardWidth := lipgloss.Width(listCardRendered)
			listContentY := listY + 4

			if mouse.X >= listX && mouse.X < listX+listCardWidth && mouse.Y >= listContentY {
				row := mouse.Y - listContentY
				pageStart := m.currencyList.Index() - m.currencyList.Cursor()
				itemIndex := pageStart + row
				visibleItems := m.currencyList.VisibleItems()
				if itemIndex >= 0 && itemIndex < len(visibleItems) {
					return func() tea.Msg {
						return currencyClickedMsg{target: "converter", index: itemIndex}
					}
				}
			}

			rightX := bodyX
			rightY := bodyY
			if !stacked {
				rightX = listX + listCardWidth + 1
			} else {
				rightY = bodyY + lipgloss.Height(listCardRendered)
			}

			contentStartY := rightY + 2
			if hitRect(mouse.X, mouse.Y, rightX+2, contentStartY+4, maxInt(rightWidth-4, 18), 1) {
				return func() tea.Msg { return mouseActionMsg{action: "focus-amount"} }
			}
			if hitRect(mouse.X, mouse.Y, rightX+2, contentStartY+8, maxInt(rightWidth-4, 18), 1) {
				return func() tea.Msg { return mouseActionMsg{action: "focus-foreign-amount"} }
			}
			if hitRect(mouse.X, mouse.Y, rightX+2, contentStartY+12, maxInt(rightWidth-4, 18), 1) {
				return func() tea.Msg { return mouseActionMsg{action: "focus-date"} }
			}
			if hitRect(mouse.X, mouse.Y, rightX+2, contentStartY+14, maxInt(rightWidth-4, 18), 2) {
				return func() tea.Msg { return mouseActionMsg{action: "convert"} }
			}
			return nil
		}

		if m.activeTab == 2 {
			leftWidth, _, _ := splitPaneWidths(bodyWidth, 28, 32)
			leftX := bodyX
			leftY := bodyY
			if hitRect(mouse.X, mouse.Y, leftX+2, leftY+3, maxInt(leftWidth-4, 18), 1) {
				return func() tea.Msg { return mouseActionMsg{action: "focus-db-filter"} }
			}
			if hitRect(mouse.X, mouse.Y, leftX+2, leftY+6, maxInt(leftWidth-4, 18), 1) {
				return func() tea.Msg { return mouseActionMsg{action: "toggle-db-sort"} }
			}
			listTopY := leftY + 10
			firstItemY := listTopY + 4
			if hitRect(mouse.X, mouse.Y, leftX+1, firstItemY, leftWidth-2, maxInt(m.bodyAreaHeight()-18, 4)) {
				row := mouse.Y - firstItemY
				pageStart := m.dbCurrencyList.Index() - m.dbCurrencyList.Cursor()
				itemIndex := pageStart + row/2
				visibleItems := m.dbCurrencyList.VisibleItems()
				if itemIndex >= 0 && itemIndex < len(visibleItems) {
					return func() tea.Msg { return currencyClickedMsg{target: "db", index: itemIndex} }
				}
			}
			return nil
		}

		formX := bodyX
		formContentY := bodyY + 2
		cacheFormWidth, _, _ := splitPaneWidths(bodyWidth, 34, 32)
		if hitRect(mouse.X, mouse.Y, formX+2, formContentY+2, maxInt(cacheFormWidth-4, 18), 1) {
			return func() tea.Msg { return mouseActionMsg{action: "focus-cache-from"} }
		}
		if hitRect(mouse.X, mouse.Y, formX+2, formContentY+5, maxInt(cacheFormWidth-4, 18), 1) {
			return func() tea.Msg { return mouseActionMsg{action: "focus-cache-to"} }
		}
		if hitRect(mouse.X, mouse.Y, formX+2, formContentY+12, maxInt(cacheFormWidth-4, 18), 1) {
			return func() tea.Msg { return mouseActionMsg{action: "prefetch-cache"} }
		}
		return nil
	}
}

func (m *tuiModel) startRangePrefetch() (tea.Model, tea.Cmd) {
	startDate, err := ParseDate(m.cacheFromInput.Value())
	if err != nil {
		m.lastError = err.Error()
		return m, nil
	}
	endDate, err := ParseDate(m.cacheToInput.Value())
	if err != nil {
		m.lastError = err.Error()
		return m, nil
	}

	m.cfg.LastFromDate = m.cacheFromInput.Value()

	currencies := m.selectedCacheCurrencyCodes()
	if len(currencies) == 0 {
		m.lastError = "lista walut nie jest jeszcze gotowa"
		return m, nil
	}

	chunks := buildPrefetchChunks(currencies, startDate, endDate)
	if len(chunks) == 0 {
		m.lastError = "brak danych do pobrania w podanym zakresie"
		return m, nil
	}

	m.cacheBusy = true
	m.lastError = ""
	m.prefetchChunks = nil
	m.prefetchDone = 0
	m.prefetchSummary = nbp.PrefetchSummary{
		CurrencyCount: len(currencies),
		RateCount:     0,
		StartDate:     startDate.Format("2006-01-02"),
		EndDate:       endDate.Format("2006-01-02"),
	}
	m.status = fmt.Sprintf("Pobieranie zakresu %s -> %s do bazy cache...", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	m.prefetchChunks = chunks[1:]
	cmds := []tea.Cmd{
		m.progress.SetPercent(0),
		prefetchChunkCmd(m.service, m.cfg.TimeoutSeconds, chunks[0]),
		saveConfigCmd(m.configPath, m.cfg),
		spinnerTickCmd(m.spinner),
	}
	return m, tea.Batch(cmds...)
}

func (m *tuiModel) refreshCacheViewport() {
	width := m.cacheViewport.Width()
	if width <= 0 {
		width = 60
	}

	lines := []string{renderCacheInfo(m.cacheInfo, width-2)}
	if m.prefetchSummary.CurrencyCount > 0 {
		lines = append(lines,
			"",
			sectionTitle("OSTATNI IMPORT ZAKRESU"),
			fmt.Sprintf("Walut: %d", m.prefetchSummary.CurrencyCount),
			fmt.Sprintf("Kursów: %d", m.prefetchSummary.RateCount),
			fmt.Sprintf("Zakres: %s -> %s", m.prefetchSummary.StartDate, m.prefetchSummary.EndDate),
		)
	}

	m.cacheViewport.SetContent(strings.Join(lines, "\n"))
}

func (m *tuiModel) refreshDBViewport() {
	stat, ok := m.selectedDBStat()
	if !ok {
		m.dbViewport.SetContent("Wybierz walutę z listy, aby zobaczyć historię kursów.")
		return
	}

	lines := []string{
		sectionTitle("SZCZEGÓŁY WALUTY"),
		fmt.Sprintf("Kod: %s", stat.Code),
		fmt.Sprintf("Nazwa: %s", stat.Name),
		fmt.Sprintf("Liczba kursów: %d", stat.RateCount),
		fmt.Sprintf("Pierwszy dzień: %s", orDash(stat.FirstDate)),
		fmt.Sprintf("Ostatni dzień: %s", orDash(stat.LastDate)),
		"",
		sectionTitle("HISTORIA KURSÓW"),
	}
	if len(m.selectedDBHistory) == 0 {
		lines = append(lines, "Brak historii kursów dla wybranej waluty.")
		m.dbViewport.SetContent(strings.Join(lines, "\n"))
		return
	}

	for _, entry := range m.selectedDBHistory {
		lines = append(lines,
			fmt.Sprintf("%s | %.4f | %s", entry.EffectiveRateDate, entry.Mid, orDash(entry.TableNo)),
		)
	}
	m.dbViewport.SetContent(strings.Join(lines, "\n"))
}

func (m dbSortMode) next() dbSortMode {
	switch m {
	case dbSortByCode:
		return dbSortByRateCount
	case dbSortByRateCount:
		return dbSortByLastDate
	default:
		return dbSortByCode
	}
}

func (m dbSortMode) label() string {
	switch m {
	case dbSortByRateCount:
		return "WG LICZBY KURSÓW"
	case dbSortByLastDate:
		return "WG OSTATNIEJ DATY"
	default:
		return "WG KODU"
	}
}

func loadCurrenciesCmd(service *nbp.Service, timeoutSeconds int) tea.Cmd {
	return func() tea.Msg {
		timeout := time.Duration(timeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		currencies, err := service.GetCurrencies(ctx)
		return currenciesLoadedMsg{currencies: currencies, err: err}
	}
}

func prefetchChunkCmd(service *nbp.Service, timeoutSeconds int, chunk prefetchChunk) tea.Cmd {
	return func() tea.Msg {
		timeout := time.Duration(timeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		count, err := service.ImportRateRangeChunk(ctx, chunk.currency, chunk.start, chunk.end)
		return prefetchChunkFinishedMsg{chunk: chunk, rateCount: count, err: err}
	}
}

func convertCurrencyCmd(service *nbp.Service, timeoutSeconds int, currency models.Currency, amountInput, dateInput string, direction conversionDirection) tea.Cmd {
	return func() tea.Msg {
		amount, err := ParseAmount(amountInput)
		if err != nil {
			return convertFinishedMsg{err: err}
		}
		requestedDate, err := ParseDate(dateInput)
		if err != nil {
			return convertFinishedMsg{err: err}
		}

		timeout := time.Duration(timeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		rate, err := service.GetRate(ctx, currency.Code, requestedDate)
		if err != nil {
			return convertFinishedMsg{err: err}
		}

		converted := amount / rate.Mid
		baseCurrency := "PLN"
		targetCurrency := currency.Code
		if direction == directionForeignToPLN {
			converted = amount * rate.Mid
			baseCurrency = currency.Code
			targetCurrency = "PLN"
		}

		sourceBox := lipgloss.NewStyle().
			Bold(true).
			Padding(1, 2).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("24")).
			Render(fmt.Sprintf("%.4f %s", amount, baseCurrency))

		resultBox := lipgloss.NewStyle().
			Bold(true).
			Padding(1, 2).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("28")).
			Render(fmt.Sprintf("%.4f %s", converted, targetCurrency))

		result := strings.Join([]string{
			sectionTitle("PRZELICZENIE"),
			sourceBox,
			resultBox,
			"",
			sectionTitle("UŻYTY KURS"),
			fmt.Sprintf("Waluta: %s", rate.Currency),
			fmt.Sprintf("Data żądana: %s", rate.RequestedDate),
			fmt.Sprintf("Data kursu: %s", rate.EffectiveRateDate),
			fmt.Sprintf("Kurs średni NBP: %.4f", rate.Mid),
			fmt.Sprintf("Tabela: %s", orDash(rate.TableNo)),
			fmt.Sprintf("Źródło: %s", rate.Source),
		}, "\n")

		return convertFinishedMsg{
			result:       result,
			direction:    direction,
			sourceAmount: amount,
			targetAmount: converted,
		}
	}
}

func loadCacheInfoCmd(store cache.Store) tea.Cmd {
	return func() tea.Msg {
		info, err := store.Info()
		return cacheInfoLoadedMsg{info: info, err: err}
	}
}

func loadCurrencyStatsCmd(store cache.Store) tea.Cmd {
	return func() tea.Msg {
		stats, err := store.ListCurrencyStats()
		return currencyStatsLoadedMsg{stats: stats, err: err}
	}
}

func loadCurrencyHistoryCmd(store cache.Store, currency string, limit int) tea.Cmd {
	return func() tea.Msg {
		history, err := store.ListCurrencyHistory(currency, limit)
		return currencyHistoryLoadedMsg{currency: currency, history: history, err: err}
	}
}

func clearCacheCmd(store cache.Store) tea.Cmd {
	return func() tea.Msg {
		err := store.Clear()
		return cacheClearedMsg{err: err}
	}
}

func saveConfigCmd(configPath string, cfg models.AppConfig) tea.Cmd {
	return func() tea.Msg {
		return configSavedMsg{err: SaveConfigAtPath(configPath, cfg)}
	}
}

func spinnerTickCmd(model spinner.Model) tea.Cmd {
	return func() tea.Msg {
		return model.Tick()
	}
}

func renderCacheInfo(info cache.Info, width int) string {
	pathLines := wrapPath(info.Path, maxInt(width, 24))
	return strings.Join([]string{
		sectionTitle("PLIK BAZY"),
		pathLines,
		"",
		sectionTitle("STATYSTYKI"),
		fmt.Sprintf("Zapisanych kursów: %d", info.Entries),
		fmt.Sprintf("Mapowań zapytań: %d", info.QueryMappings),
		fmt.Sprintf("Walut w bazie: %d", info.CurrencyCount),
		fmt.Sprintf("Rozmiar pliku: %d B", info.SizeBytes),
		fmt.Sprintf("Ostatni zapis: %s", orDash(info.LastSavedAt)),
		"",
		sectionTitle("AKCJE"),
		"tab / shift+tab - przechodzenie między polami",
		"enter - uruchom import dla całego zakresu",
		"r - odśwież statystyki",
		"c - wyczyść całą bazę cache",
	}, "\n")
}

func buildPrefetchChunks(currencies []string, startDate, endDate time.Time) []prefetchChunk {
	const chunkDays = 90
	chunks := make([]prefetchChunk, 0, len(currencies)*4)
	for _, currency := range currencies {
		for chunkStart := startDate; !chunkStart.After(endDate); chunkStart = chunkStart.AddDate(0, 0, chunkDays+1) {
			chunkEnd := chunkStart.AddDate(0, 0, chunkDays)
			if chunkEnd.After(endDate) {
				chunkEnd = endDate
			}
			chunks = append(chunks, prefetchChunk{
				currency: currency,
				start:    chunkStart,
				end:      chunkEnd,
			})
		}
	}
	return chunks
}

func wrapPath(path string, width int) string {
	if width <= 0 || len(path) <= width {
		return path
	}

	replacer := strings.NewReplacer("\\", "\\|", "/", "/|")
	parts := strings.Split(replacer.Replace(path), "|")
	lines := make([]string, 0, len(parts))
	current := ""

	for _, part := range parts {
		if current == "" {
			current = part
			continue
		}
		if len(current)+len(part) <= width {
			current += part
			continue
		}
		lines = append(lines, current)
		current = part
	}
	if current != "" {
		lines = append(lines, current)
	}

	return strings.Join(lines, "\n")
}

func batchCmds(cmds ...tea.Cmd) tea.Cmd {
	filtered := make([]tea.Cmd, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd != nil {
			filtered = append(filtered, cmd)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return tea.Batch(filtered...)
}

func (m *tuiModel) syncCacheCurrencyList() tea.Cmd {
	items := make([]list.Item, 0, len(m.currencies))
	for _, currency := range m.currencies {
		items = append(items, cacheCurrencyItem{
			code:     currency.Code,
			name:     currency.Name,
			selected: m.selectedCacheCurrencies[currency.Code],
		})
	}
	cmd := m.cacheCurrencyList.SetItems(items)
	if len(items) > 0 && m.cacheCurrencyList.Index() >= len(items) {
		m.cacheCurrencyList.Select(len(items) - 1)
	}
	return cmd
}

func (m *tuiModel) rebuildDBCurrencyList() {
	filter := strings.ToLower(strings.TrimSpace(m.dbFilterInput.Value()))
	stats := make([]cache.CurrencyStat, 0, len(m.currencyStats))
	for _, stat := range m.currencyStats {
		if filter == "" || strings.Contains(strings.ToLower(stat.Code+" "+stat.Name), filter) {
			stats = append(stats, stat)
		}
	}

	switch m.dbSortMode {
	case dbSortByRateCount:
		sort.SliceStable(stats, func(i, j int) bool {
			if stats[i].RateCount == stats[j].RateCount {
				return stats[i].Code < stats[j].Code
			}
			return stats[i].RateCount > stats[j].RateCount
		})
	case dbSortByLastDate:
		sort.SliceStable(stats, func(i, j int) bool {
			if stats[i].LastDate == stats[j].LastDate {
				return stats[i].Code < stats[j].Code
			}
			return stats[i].LastDate > stats[j].LastDate
		})
	default:
		sort.SliceStable(stats, func(i, j int) bool {
			return stats[i].Code < stats[j].Code
		})
	}

	m.filteredCurrencyStats = stats
	items := make([]list.Item, 0, len(stats))
	selectedIndex := 0
	for i, stat := range stats {
		items = append(items, dbCurrencyItem{stat: stat})
		if strings.EqualFold(stat.Code, m.selectedDBCurrency) {
			selectedIndex = i
		}
	}
	_ = m.dbCurrencyList.SetItems(items)

	if len(items) == 0 {
		m.dbCurrencyList.ResetSelected()
		m.selectedDBCurrency = ""
		m.selectedDBHistory = nil
		m.refreshDBViewport()
		return
	}

	m.dbCurrencyList.Select(selectedIndex)
	selected, ok := m.selectedDBStat()
	if ok && !strings.EqualFold(selected.Code, m.selectedDBCurrency) {
		m.selectedDBCurrency = selected.Code
		m.selectedDBHistory = nil
	}
	m.refreshDBViewport()
}

func (m tuiModel) selectedDBStat() (cache.CurrencyStat, bool) {
	item, ok := m.dbCurrencyList.SelectedItem().(dbCurrencyItem)
	if !ok {
		return cache.CurrencyStat{}, false
	}
	return item.stat, true
}

func (m *tuiModel) loadSelectedDBCurrencyHistory() (tea.Model, tea.Cmd) {
	stat, ok := m.selectedDBStat()
	if !ok {
		return m, nil
	}
	m.selectedDBCurrency = stat.Code
	m.status = fmt.Sprintf("Ładowanie historii waluty %s...", stat.Code)
	return m, loadCurrencyHistoryCmd(m.store, stat.Code, 120)
}

func (m *tuiModel) toggleCurrentCacheCurrency() tea.Cmd {
	item, ok := m.cacheCurrencyList.SelectedItem().(cacheCurrencyItem)
	if !ok {
		return nil
	}
	code := strings.ToUpper(item.code)
	if m.selectedCacheCurrencies[code] {
		delete(m.selectedCacheCurrencies, code)
	} else {
		m.selectedCacheCurrencies[code] = true
	}
	cmd := m.syncCacheCurrencyList()
	m.status = m.cacheSelectionStatus()
	return cmd
}

func (m tuiModel) selectedCacheCurrencyCodes() []string {
	codes := make([]string, 0, len(m.currencies))
	for _, currency := range m.currencies {
		codes = append(codes, currency.Code)
	}
	return codes
}

func (m tuiModel) cacheAllLabel() string {
	if m.allCacheCurrencies {
		return "[x] WSZYSTKIE WALUTY"
	}
	return "[ ] WSZYSTKIE WALUTY"
}

func (m tuiModel) cacheSelectionStatus() string {
	if m.allCacheCurrencies {
		return fmt.Sprintf("Wybrane wszystkie waluty: %d", len(m.currencies))
	}
	return fmt.Sprintf("Wybranych walut: %d", len(m.selectedCacheCurrencies))
}

func (m tuiModel) cacheSelectionPreview() string {
	if m.allCacheCurrencies {
		return "Zakres obejmie wszystkie waluty z tabeli A."
	}
	if len(m.selectedCacheCurrencies) == 0 {
		return "Nie wybrano jeszcze żadnej waluty."
	}

	codes := make([]string, 0, len(m.selectedCacheCurrencies))
	for _, currency := range m.currencies {
		if m.selectedCacheCurrencies[currency.Code] {
			codes = append(codes, currency.Code)
		}
	}
	if len(codes) > 6 {
		return "Wybrane: " + strings.Join(codes[:6], ", ") + " ..."
	}
	return "Wybrane: " + strings.Join(codes, ", ")
}

func (m tuiModel) cachePercent() float64 {
	total := len(m.prefetchChunks) + m.prefetchDone
	if total == 0 {
		return 0
	}
	return float64(m.prefetchDone) / float64(total)
}

func cardStyle(focused bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(1, 1)
	if focused {
		return style.BorderForeground(lipgloss.Color("12"))
	}
	return style.BorderForeground(lipgloss.Color("8"))
}

func shellStyle(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(1, 1).
		Width(width).
		Height(height)
}

func (m tuiModel) shellInnerWidth() int {
	width := m.width - 4
	if width < 24 {
		return 24
	}
	return width
}

func (m tuiModel) bodyAreaHeight() int {
	innerWidth := maxInt(m.width-4, 10)
	innerHeight := maxInt(m.height-4, 10)
	headerHeight := lipgloss.Height(m.renderHeader(innerWidth))
	statusHeight := lipgloss.Height(m.renderStatus(innerWidth))
	footerHeight := lipgloss.Height(m.renderFooter(innerWidth))
	height := innerHeight - headerHeight - statusHeight - footerHeight - 2
	if height < 3 {
		return 3
	}
	return height
}

func hitRect(x, y, left, top, width, height int) bool {
	return x >= left && x < left+width && y >= top && y < top+height
}

func sectionTitle(label string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("11")).
		Render(label)
}

func primaryButtonStyle(active bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		Foreground(lipgloss.Color("230"))
	if active {
		return style.Background(lipgloss.Color("28")).BorderForeground(lipgloss.Color("114"))
	}
	return style.Background(lipgloss.Color("236")).BorderForeground(lipgloss.Color("241"))
}

func secondaryButtonStyle(active bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		Foreground(lipgloss.Color("230"))
	if active {
		return style.Background(lipgloss.Color("24")).BorderForeground(lipgloss.Color("81"))
	}
	return style.Background(lipgloss.Color("237")).BorderForeground(lipgloss.Color("240"))
}

func (d conversionDirection) next() conversionDirection {
	if d == directionPLNToForeign {
		return directionForeignToPLN
	}
	return directionPLNToForeign
}

func (d conversionDirection) label(code string) string {
	if code == "" {
		code = "WALUTA"
	}
	if d == directionPLNToForeign {
		return "PLN -> " + strings.ToUpper(code)
	}
	return strings.ToUpper(code) + " -> PLN"
}

type converterHelpMap struct {
	keys tuiKeyMap
}

func (m converterHelpMap) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.Search, m.keys.ToggleDir, m.keys.NextFocus, m.keys.Quit}
}

func (m converterHelpMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.PrevTab, m.keys.NextTab, m.keys.NextFocus, m.keys.PrevFocus},
		{m.keys.Search, m.keys.ToggleDir, m.keys.ToggleHelp, m.keys.Quit},
	}
}

type cacheHelpMap struct {
	keys tuiKeyMap
}

func (m cacheHelpMap) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.Refresh, m.keys.ClearCache, m.keys.NextFocus, m.keys.Quit}
}

func (m cacheHelpMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.PrevTab, m.keys.NextTab, m.keys.NextFocus, m.keys.PrevFocus},
		{m.keys.Refresh, m.keys.ClearCache},
		{m.keys.ToggleHelp, m.keys.Quit},
	}
}

type databaseHelpMap struct {
	keys tuiKeyMap
}

func (m databaseHelpMap) ShortHelp() []key.Binding {
	return []key.Binding{m.keys.Search, m.keys.Refresh, m.keys.NextFocus, m.keys.Quit}
}

func (m databaseHelpMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.PrevTab, m.keys.NextTab, m.keys.NextFocus, m.keys.PrevFocus},
		{m.keys.Search, m.keys.Refresh},
		{m.keys.ToggleHelp, m.keys.Quit},
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func splitPaneWidths(innerWidth, minLeft, minRight int) (int, int, bool) {
	if innerWidth < minLeft+minRight+1 {
		return innerWidth, innerWidth, true
	}

	left := innerWidth / 3
	if left < minLeft {
		left = minLeft
	}
	if left > innerWidth-minRight-1 {
		left = innerWidth - minRight - 1
	}
	right := innerWidth - left - 1
	if right < minRight {
		return innerWidth, innerWidth, true
	}
	return left, right, false
}

func formatAmount(value float64) string {
	return fmt.Sprintf("%.4f", value)
}
