package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"kursomat/internal/cache"
	"kursomat/internal/models"
	"kursomat/internal/nbp"
)

type conversionDirection int

const (
	directionPLNToForeign conversionDirection = iota
	directionForeignToPLN
)

type currenciesLoadedMsg struct {
	currencies []models.Currency
	err        error
}

type currencyClickedMsg struct {
	index int
}

type convertFinishedMsg struct {
	result string
	err    error
}

type cacheInfoLoadedMsg struct {
	info cache.Info
	err  error
}

type cacheClearedMsg struct {
	err error
}

type currencyItem struct {
	code string
	name string
}

func (c currencyItem) Title() string       { return strings.ToUpper(c.code) + "  " + c.name }
func (c currencyItem) Description() string { return c.name }
func (c currencyItem) FilterValue() string { return c.code + " " + c.name }

type tuiKeyMap struct {
	PrevTab    key.Binding
	NextTab    key.Binding
	NextFocus  key.Binding
	PrevFocus  key.Binding
	Search     key.Binding
	ToggleDir  key.Binding
	Refresh    key.Binding
	ClearCache key.Binding
	ToggleHelp key.Binding
	Quit       key.Binding
}

func defaultTUIKeyMap() tuiKeyMap {
	return tuiKeyMap{
		PrevTab:    key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "poprzednia zakładka")),
		NextTab:    key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "następna zakładka")),
		NextFocus:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "następny fokus")),
		PrevFocus:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "poprzedni fokus")),
		Search:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "szukaj waluty")),
		ToggleDir:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "odwróć kierunek")),
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
	cfg     models.AppConfig
	service *nbp.Service
	store   cache.Store

	keys    tuiKeyMap
	help    help.Model
	spinner spinner.Model

	width  int
	height int

	activeTab int
	focus     int

	currencyList list.Model
	amountInput  textinput.Model
	dateInput    textinput.Model

	resultsViewport viewport.Model
	cacheViewport   viewport.Model

	direction       conversionDirection
	loading         bool
	cacheBusy       bool
	lastError       string
	status          string
	showHelp        bool
	currenciesReady bool
	cacheInfo       cache.Info
}

func newTUIModel(cfg models.AppConfig, service *nbp.Service, store cache.Store) tuiModel {
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

	amountInput := textinput.New()
	amountInput.Prompt = ""
	amountInput.Placeholder = "np. 1500.00"
	amountInput.CharLimit = 20
	amountInput.SetValue("100.00")
	amountInput.SetWidth(24)
	amountInput.SetVirtualCursor(true)

	dateInput := textinput.New()
	dateInput.Prompt = ""
	dateInput.Placeholder = "YYYY-MM-DD"
	dateInput.SetValue(time.Now().Format("2006-01-02"))
	dateInput.SetWidth(24)
	dateInput.SetVirtualCursor(true)

	resultsViewport := viewport.New(viewport.WithWidth(60), viewport.WithHeight(14))
	resultsViewport.SetContent("WYBIERZ WALUTĘ Z PICKERA, WPISZ KWOTĘ I DATĘ, A NASTĘPNIE NACIŚNIJ [ PRZELICZ ].")

	cacheViewport := viewport.New(viewport.WithWidth(60), viewport.WithHeight(14))
	cacheViewport.SetContent("ŁADOWANIE INFORMACJI O BAZIE CACHE...")

	helpModel := help.New()
	helpModel.ShowAll = false

	spin := spinner.New(spinner.WithSpinner(spinner.Line))
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)

	return tuiModel{
		cfg:             cfg,
		service:         service,
		store:           store,
		keys:            defaultTUIKeyMap(),
		help:            helpModel,
		spinner:         spin,
		currencyList:    currencyList,
		amountInput:     amountInput,
		dateInput:       dateInput,
		resultsViewport: resultsViewport,
		cacheViewport:   cacheViewport,
		status:          "Gotowy",
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(
		m.amountInput.Focus(),
		textinput.Blink,
		loadCurrenciesCmd(m.service, m.cfg.TimeoutSeconds),
		loadCacheInfoCmd(m.store),
	)
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.loading || m.cacheBusy {
		var spinCmd tea.Cmd
		m.spinner, spinCmd = m.spinner.Update(msg)
		if spinCmd != nil {
			defer func(existing tea.Cmd) {
				if existing != nil {
					spinCmd = tea.Batch(existing, spinnerTickCmd(m.spinner))
				} else {
					spinCmd = spinnerTickCmd(m.spinner)
				}
			}(spinCmd)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil

	case currencyClickedMsg:
		m.focus = 0
		m.currencyList.Select(msg.index)
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case currenciesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.lastError = humanizeError(msg.err)
			m.status = "Nie udało się załadować listy walut"
			return m, nil
		}
		m.lastError = ""
		m.currenciesReady = true
		items := make([]list.Item, 0, len(msg.currencies))
		selectedIndex := 0
		for i, currency := range msg.currencies {
			items = append(items, currencyItem{code: currency.Code, name: currency.Name})
			if strings.EqualFold(currency.Code, "USD") {
				selectedIndex = i
			}
		}
		cmd := m.currencyList.SetItems(items)
		m.currencyList.Select(selectedIndex)
		m.status = fmt.Sprintf("Załadowano %d walut NBP", len(msg.currencies))
		return m, tea.Batch(cmd)

	case convertFinishedMsg:
		m.loading = false
		if msg.err != nil {
			m.lastError = humanizeError(msg.err)
			m.status = "Błąd przeliczenia"
			return m, nil
		}
		m.lastError = ""
		m.status = "Przeliczenie zakończone"
		m.resultsViewport.SetContent(msg.result)
		m.resultsViewport.GotoTop()
		return m, loadCacheInfoCmd(m.store)

	case cacheInfoLoadedMsg:
		m.cacheBusy = false
		if msg.err != nil {
			m.lastError = humanizeError(msg.err)
			m.status = "Błąd odczytu bazy cache"
			return m, nil
		}
		m.lastError = ""
		m.cacheInfo = msg.info
		m.cacheViewport.SetContent(renderCacheInfo(msg.info))
		if m.activeTab == 1 {
			m.status = "Odczytano statystyki bazy cache"
		}
		return m, nil

	case cacheClearedMsg:
		m.cacheBusy = false
		if msg.err != nil {
			m.lastError = humanizeError(msg.err)
			m.status = "Błąd czyszczenia bazy cache"
			return m, nil
		}
		m.lastError = ""
		m.status = "Baza cache wyczyszczona"
		m.resultsViewport.SetContent("CACHE ZOSTAŁ WYCZYSZCZONY. NOWE KURSY BĘDĄ DOPISYWANE PRZY KOLEJNYCH ZAPYTANIACH.")
		return m, tea.Batch(loadCacheInfoCmd(m.store), loadCurrenciesCmd(m.service, m.cfg.TimeoutSeconds))
	}

	if m.activeTab == 0 {
		return m.updateConverterComponents(msg)
	}
	return m.updateCacheComponents(msg)
}

func (m tuiModel) View() tea.View {
	header := m.renderHeader()
	body := m.renderBody()
	status := m.renderStatus()
	footer := m.renderFooter()

	root := lipgloss.NewStyle().
		Padding(1, 2).
		Width(maxInt(m.width, 80)).
		Height(maxInt(m.height, 24))

	v := tea.NewView(root.Render(lipgloss.JoinVertical(lipgloss.Left, header, body, status, footer)))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.OnMouse = m.onMouse()
	return v
}

func (m *tuiModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.activeTab == 0 && m.focus == 0 && m.currencyList.SettingFilter() {
		return m.handleConverterKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.PrevTab) && !m.converterInputFocused():
		m.activeTab = 0
		m.focus = 0
		return m, m.focusCurrentField()
	case key.Matches(msg, m.keys.NextTab) && !m.converterInputFocused():
		m.activeTab = 1
		m.focus = 0
		return m, loadCacheInfoCmd(m.store)
	case key.Matches(msg, m.keys.ToggleHelp):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return m, nil
	}

	if m.activeTab == 0 {
		return m.handleConverterKey(msg)
	}
	return m.handleCacheKey(msg)
}

func (m *tuiModel) handleConverterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focus = (m.focus + 1) % 5
		return m, m.focusCurrentField()
	case "shift+tab":
		m.focus = (m.focus + 4) % 5
		return m, m.focusCurrentField()
	case "enter":
		if m.focus == 3 {
			m.direction = m.direction.next()
			return m, nil
		}
		if m.focus == 4 && !m.loading {
			return m.startConversion()
		}
		if m.focus == 1 || m.focus == 2 {
			m.focus = (m.focus + 1) % 5
			return m, m.focusCurrentField()
		}
	case " ":
		if m.focus == 3 {
			m.direction = m.direction.next()
			return m, nil
		}
	}

	if key.Matches(msg, m.keys.ToggleDir) {
		m.direction = m.direction.next()
		return m, nil
	}

	return m.updateConverterComponents(msg)
}

func (m *tuiModel) handleCacheKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
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

func (m *tuiModel) updateConverterComponents(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if m.focus == 0 {
		m.currencyList, cmd = m.currencyList.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.focus == 1 {
		m.amountInput, cmd = m.amountInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.focus == 2 {
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

func (m *tuiModel) updateCacheComponents(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.cacheViewport, cmd = m.cacheViewport.Update(msg)
	return m, cmd
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
	m.loading = true
	m.lastError = ""
	m.status = "Pobieranie kursu i przeliczanie kwoty..."
	return m, tea.Batch(
		convertCurrencyCmd(m.service, m.cfg.TimeoutSeconds, currency, m.amountInput.Value(), m.dateInput.Value(), m.direction),
		spinnerTickCmd(m.spinner),
	)
}

func (m *tuiModel) focusCurrentField() tea.Cmd {
	m.amountInput.Blur()
	m.dateInput.Blur()

	switch m.focus {
	case 1:
		return m.amountInput.Focus()
	case 2:
		return m.dateInput.Focus()
	default:
		return nil
	}
}

func (m *tuiModel) resize() {
	innerWidth := maxInt(m.width-8, 72)
	innerHeight := maxInt(m.height-10, 16)
	listWidth := maxInt(innerWidth/3, 28)
	listHeight := maxInt(innerHeight, 12)
	rightWidth := maxInt(innerWidth-listWidth-3, 36)

	m.currencyList.SetSize(listWidth, listHeight)
	m.amountInput.SetWidth(maxInt(rightWidth-8, 18))
	m.dateInput.SetWidth(maxInt(rightWidth-8, 18))
	m.resultsViewport.SetWidth(rightWidth - 4)
	m.resultsViewport.SetHeight(maxInt(innerHeight-13, 8))
	m.cacheViewport.SetWidth(innerWidth - 4)
	m.cacheViewport.SetHeight(maxInt(innerHeight-4, 10))
	m.help.SetWidth(innerWidth)
}

func (m tuiModel) renderHeader() string {
	tabBase := lipgloss.NewStyle().
		Padding(0, 2).
		Bold(true)
	active := tabBase.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("25"))
	inactive := tabBase.Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238"))

	converterTab := inactive.Render("KONWERTER")
	cacheTab := inactive.Render("BAZA CACHE")
	if m.activeTab == 0 {
		converterTab = active.Render("KONWERTER")
	} else {
		cacheTab = active.Render("BAZA CACHE")
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("18")).
		Padding(0, 2).
		Render("KURSOWNIK NBP")

	return lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", converterTab, " ", cacheTab)
}

func (m tuiModel) renderBody() string {
	if m.activeTab == 0 {
		return m.renderConverterBody()
	}
	return m.renderCacheBody()
}

func (m tuiModel) renderConverterBody() string {
	listCard := cardStyle(m.focus == 0).Render(m.currencyList.View())

	selectedCode := "?"
	selectedName := "brak wybranej waluty"
	if currency, ok := m.selectedCurrency(); ok {
		selectedCode = currency.Code
		selectedName = currency.Name
	}

	directionButton := secondaryButtonStyle(m.focus == 3).Render(m.direction.label(selectedCode))
	convertButtonLabel := "[ PRZELICZ ]"
	if m.loading {
		convertButtonLabel = m.spinner.View() + " PRZELICZANIE"
	}
	convertButton := primaryButtonStyle(m.focus == 4 && !m.loading).Render(convertButtonLabel)

	resultTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")).
		Background(lipgloss.Color("18")).
		Padding(0, 1).
		Render(strings.ToUpper("wynik i kurs z dnia"))

	formLines := []string{
		sectionTitle("AKTYWNA WALUTA"),
		lipgloss.NewStyle().Bold(true).Render(selectedCode + "  " + selectedName),
		"",
		sectionTitle("KWOTA"),
		m.amountInput.View(),
		"",
		sectionTitle("DZIEŃ KURSU (RRRR-MM-DD)"),
		m.dateInput.View(),
		"",
		sectionTitle("KIERUNEK"),
		directionButton,
		"",
		convertButton,
		"",
		resultTitle,
		m.resultsViewport.View(),
	}
	rightCard := cardStyle(m.focus >= 1).Render(strings.Join(formLines, "\n"))

	if m.width < 110 {
		return lipgloss.JoinVertical(lipgloss.Left, listCard, rightCard)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, listCard, " ", rightCard)
}

func (m tuiModel) renderCacheBody() string {
	body := m.cacheViewport.View()
	if m.cacheBusy {
		body = m.spinner.View() + " AKTUALIZACJA BAZY CACHE\n\n" + body
	}
	return cardStyle(false).Render(body)
}

func (m tuiModel) renderStatus() string {
	style := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236"))

	if m.lastError != "" {
		return style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("1")).Render("BŁĄD: " + strings.ToUpper(m.lastError))
	}
	return style.Render("STATUS: " + strings.ToUpper(m.status))
}

func (m tuiModel) renderFooter() string {
	if m.activeTab == 1 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Render(
			m.help.View(cacheHelpMap{keys: m.keys}),
		)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Render(
		m.help.View(converterHelpMap{keys: m.keys}),
	)
}

func (m tuiModel) selectedCurrency() (models.Currency, bool) {
	item, ok := m.currencyList.SelectedItem().(currencyItem)
	if !ok {
		return models.Currency{}, false
	}
	return models.Currency{Code: item.code, Name: item.name}, true
}

func (m tuiModel) converterInputFocused() bool {
	return m.activeTab == 0 && (m.focus == 1 || m.focus == 2)
}

func (m tuiModel) onMouse() func(msg tea.MouseMsg) tea.Cmd {
	headerHeight := lipgloss.Height(m.renderHeader())
	rootPaddingTop := 1
	rootPaddingLeft := 2

	return func(msg tea.MouseMsg) tea.Cmd {
		if m.activeTab != 0 || m.focus == 1 || m.focus == 2 {
			return nil
		}

		mouse := msg.Mouse()
		if mouse.Button != tea.MouseLeft {
			return nil
		}

		listX := rootPaddingLeft
		listY := rootPaddingTop + headerHeight
		listCardWidth := lipgloss.Width(cardStyle(m.focus == 0).Render(m.currencyList.View()))
		listContentY := listY + 4

		if mouse.X < listX || mouse.X >= listX+listCardWidth {
			return nil
		}
		if mouse.Y < listContentY {
			return nil
		}

		row := mouse.Y - listContentY
		pageStart := m.currencyList.Index() - m.currencyList.Cursor()
		itemIndex := pageStart + row
		visibleItems := m.currencyList.VisibleItems()
		if itemIndex < 0 || itemIndex >= len(visibleItems) {
			return nil
		}

		return func() tea.Msg {
			return currencyClickedMsg{index: itemIndex}
		}
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

		return convertFinishedMsg{result: result}
	}
}

func loadCacheInfoCmd(store cache.Store) tea.Cmd {
	return func() tea.Msg {
		info, err := store.Info()
		return cacheInfoLoadedMsg{info: info, err: err}
	}
}

func clearCacheCmd(store cache.Store) tea.Cmd {
	return func() tea.Msg {
		err := store.Clear()
		return cacheClearedMsg{err: err}
	}
}

func spinnerTickCmd(model spinner.Model) tea.Cmd {
	return func() tea.Msg {
		return model.Tick()
	}
}

func renderCacheInfo(info cache.Info) string {
	return strings.Join([]string{
		sectionTitle("PLIK BAZY"),
		info.Path,
		"",
		sectionTitle("STATYSTYKI"),
		fmt.Sprintf("Zapisanych kursów: %d", info.Entries),
		fmt.Sprintf("Mapowań zapytań: %d", info.QueryMappings),
		fmt.Sprintf("Walut w bazie: %d", info.CurrencyCount),
		fmt.Sprintf("Rozmiar pliku: %d B", info.SizeBytes),
		fmt.Sprintf("Ostatni zapis: %s", orDash(info.LastSavedAt)),
		"",
		sectionTitle("AKCJE"),
		"/ - szukaj waluty w pickerze",
		"r - odśwież statystyki",
		"c - wyczyść całą bazę cache",
	}, "\n")
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
		Foreground(lipgloss.Color("230"))
	if active {
		return style.Background(lipgloss.Color("28"))
	}
	return style.Background(lipgloss.Color("240"))
}

func secondaryButtonStyle(active bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		Foreground(lipgloss.Color("230"))
	if active {
		return style.Background(lipgloss.Color("24"))
	}
	return style.Background(lipgloss.Color("238"))
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
	return []key.Binding{m.keys.PrevTab, m.keys.NextTab, m.keys.Refresh, m.keys.ClearCache, m.keys.Quit}
}

func (m cacheHelpMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.PrevTab, m.keys.NextTab, m.keys.Refresh, m.keys.ClearCache},
		{m.keys.ToggleHelp, m.keys.Quit},
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
