package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"kursomat/internal/cache"
	"kursomat/internal/models"
	"kursomat/internal/nbp"
)

type fetchFinishedMsg struct {
	results []models.RateResult
	err     error
}

type cacheInfoLoadedMsg struct {
	info cache.Info
	err  error
}

type cacheClearedMsg struct {
	err error
}

type tuiModel struct {
	cfg     models.AppConfig
	service *nbp.Service
	store   cache.Store

	width  int
	height int

	activeTab int
	focus     int

	currencyInput textinput.Model
	dateInput     textinput.Model

	loading   bool
	cacheBusy bool
	lastError string
	status    string

	resultsViewport viewport.Model
	cacheInfo       cache.Info
}

func newTUIModel(cfg models.AppConfig, service *nbp.Service, store cache.Store) tuiModel {
	currencyInput := textinput.New()
	currencyInput.Prompt = ""
	currencyInput.Placeholder = "USD,EUR,CHF"
	currencyInput.SetValue("USD")

	dateInput := textinput.New()
	dateInput.Prompt = ""
	dateInput.Placeholder = "YYYY-MM-DD"
	dateInput.SetValue(time.Now().Format("2006-01-02"))

	vp := viewport.New(
		viewport.WithWidth(80),
		viewport.WithHeight(12),
	)
	vp.SetContent("Brak wyników. Wpisz waluty i datę, potem naciśnij Enter na przycisku [ Pobierz ].")

	return tuiModel{
		cfg:             cfg,
		service:         service,
		store:           store,
		currencyInput:   currencyInput,
		dateInput:       dateInput,
		resultsViewport: vp,
		status:          "Gotowy",
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(
		m.currencyInput.Focus(),
		textinput.Blink,
		loadCacheInfoCmd(m.store),
	)
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "left" {
			m.activeTab = 0
			m.focus = 0
			return m, m.focusInput(0)
		}
		if msg.String() == "right" {
			m.activeTab = 1
			m.focus = 0
			return m, loadCacheInfoCmd(m.store)
		}
		if msg.String() == "?" {
			m.status = "Skróty: ←/→ zakładki, Tab fokus, Enter akcja, r odśwież cache, c wyczyść cache, q wyjście"
			return m, nil
		}

		if m.activeTab == 0 {
			return m.handleRateKey(msg)
		}
		return m.handleCacheKey(msg)

	case fetchFinishedMsg:
		m.loading = false
		if msg.err != nil {
			m.lastError = humanizeError(msg.err)
			m.status = "Błąd pobierania kursu"
			return m, nil
		}
		m.lastError = ""
		m.status = fmt.Sprintf("Pobrano %d kursów", len(msg.results))
		var out bytes.Buffer
		if err := PrintRates(&out, msg.results, models.OutputText); err != nil {
			m.lastError = humanizeError(err)
			return m, nil
		}
		m.resultsViewport.SetContent(out.String())
		m.resultsViewport.GotoTop()
		return m, nil

	case cacheInfoLoadedMsg:
		m.cacheBusy = false
		if msg.err != nil {
			m.lastError = humanizeError(msg.err)
			m.status = "Błąd odczytu cache"
			return m, nil
		}
		m.lastError = ""
		m.cacheInfo = msg.info
		m.status = "Odczytano informacje o cache"
		return m, nil

	case cacheClearedMsg:
		m.cacheBusy = false
		if msg.err != nil {
			m.lastError = humanizeError(msg.err)
			m.status = "Błąd czyszczenia cache"
			return m, nil
		}
		m.lastError = ""
		m.status = "Cache wyczyszczony"
		return m, loadCacheInfoCmd(m.store)
	}

	var cmd tea.Cmd
	if m.activeTab == 0 {
		m.resultsViewport, cmd = m.resultsViewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m tuiModel) View() tea.View {
	header := m.renderHeader()
	body := m.renderBody()
	status := m.renderStatus()
	footer := m.renderFooter()

	root := lipgloss.NewStyle().
		Padding(1, 2).
		Width(maxInt(m.width, 40)).
		Height(maxInt(m.height, 12))

	content := lipgloss.JoinVertical(lipgloss.Left, header, body, status, footer)
	v := tea.NewView(root.Render(content))
	v.AltScreen = true
	return v
}

func (m *tuiModel) handleRateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.focus = (m.focus + 1) % 3
		return m, m.focusInput(m.focus)
	case "shift+tab":
		m.focus = (m.focus + 2) % 3
		return m, m.focusInput(m.focus)
	case "enter":
		if m.focus == 2 && !m.loading {
			m.loading = true
			m.lastError = ""
			m.status = "Pobieranie kursów z NBP..."
			return m, fetchRatesCmd(m.service, m.cfg.TimeoutSeconds, m.currencyInput.Value(), m.dateInput.Value())
		}
	}

	var cmd tea.Cmd
	if m.focus == 0 {
		m.currencyInput, cmd = m.currencyInput.Update(msg)
		return m, cmd
	}
	if m.focus == 1 {
		m.dateInput, cmd = m.dateInput.Update(msg)
		return m, cmd
	}
	m.resultsViewport, cmd = m.resultsViewport.Update(msg)
	return m, cmd
}

func (m *tuiModel) handleCacheKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		if m.cacheBusy {
			return m, nil
		}
		m.cacheBusy = true
		m.status = "Odświeżanie informacji o cache..."
		return m, loadCacheInfoCmd(m.store)
	case "c":
		if m.cacheBusy {
			return m, nil
		}
		m.cacheBusy = true
		m.status = "Czyszczenie cache..."
		return m, clearCacheCmd(m.store)
	}
	return m, nil
}

func (m *tuiModel) focusInput(index int) tea.Cmd {
	m.currencyInput.Blur()
	m.dateInput.Blur()
	switch index {
	case 0:
		return m.currencyInput.Focus()
	case 1:
		return m.dateInput.Focus()
	default:
		return nil
	}
}

func (m *tuiModel) resize() {
	innerWidth := maxInt(m.width-8, 40)
	innerHeight := maxInt(m.height-10, 8)
	m.currencyInput.SetWidth(maxInt(innerWidth-4, 20))
	m.dateInput.SetWidth(maxInt(innerWidth-4, 20))
	m.resultsViewport.SetWidth(innerWidth)
	m.resultsViewport.SetHeight(maxInt(innerHeight-8, 5))
}

func (m tuiModel) renderHeader() string {
	tabActive := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render
	tabInactive := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render

	rateTab := tabInactive("Kursy")
	cacheTab := tabInactive("Cache")
	if m.activeTab == 0 {
		rateTab = tabActive("Kursy")
	} else {
		cacheTab = tabActive("Cache")
	}
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Render("Kursownik NBP  |  " + rateTab + "  •  " + cacheTab)
}

func (m tuiModel) renderBody() string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		Padding(1, 1)

	if m.activeTab == 0 {
		buttonStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("12")).
			Padding(0, 1)
		if m.focus != 2 {
			buttonStyle = buttonStyle.Background(lipgloss.Color("8"))
		}
		if m.loading {
			buttonStyle = buttonStyle.Background(lipgloss.Color("11"))
		}

		body := strings.Join([]string{
			"Waluty (np. USD,EUR,CHF):",
			m.currencyInput.View(),
			"",
			"Data (YYYY-MM-DD):",
			m.dateInput.View(),
			"",
			buttonStyle.Render("[ Pobierz ]"),
			"",
			"Wyniki:",
			m.resultsViewport.View(),
		}, "\n")
		return box.Render(body)
	}

	cacheBody := strings.Join([]string{
		fmt.Sprintf("Ścieżka: %s", orDash(m.cacheInfo.Path)),
		fmt.Sprintf("Liczba wpisów kursów: %d", m.cacheInfo.Entries),
		fmt.Sprintf("Liczba mapowań zapytań: %d", m.cacheInfo.QueryMappings),
		fmt.Sprintf("Rozmiar pliku: %d B", m.cacheInfo.SizeBytes),
		fmt.Sprintf("Ostatni zapis: %s", orDash(m.cacheInfo.LastSavedAt)),
		"",
		"Akcje:",
		"r - odśwież informacje",
		"c - wyczyść cache",
	}, "\n")
	return box.Render(cacheBody)
}

func (m tuiModel) renderStatus() string {
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(0, 1)

	if m.lastError != "" {
		return style.Foreground(lipgloss.Color("9")).Render("Błąd: " + m.lastError)
	}
	return style.Render("Status: " + m.status)
}

func (m tuiModel) renderFooter() string {
	text := "←/→ zakładki • Tab/Shift+Tab fokus • Enter akcja • r odśwież cache • c wyczyść cache • q wyjście"
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(text)
}

func fetchRatesCmd(service *nbp.Service, timeoutSeconds int, currencyInput, dateInput string) tea.Cmd {
	return func() tea.Msg {
		currencies, err := ParseCurrencies(currencyInput)
		if err != nil {
			return fetchFinishedMsg{err: err}
		}
		requestedDate, err := ParseDate(dateInput)
		if err != nil {
			return fetchFinishedMsg{err: err}
		}

		timeout := time.Duration(timeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		results, err := service.GetRates(ctx, currencies, requestedDate)
		return fetchFinishedMsg{results: results, err: err}
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
