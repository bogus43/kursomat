package cli

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"kursomat/internal/cache"
	"kursomat/internal/models"
)

func TestHandleConverterKeyTogglesManualDirection(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.lastEditedAmount = amountFieldPLN

	updatedModel, cmd := model.handleConverterKey(tea.KeyPressMsg{
		Text: "d",
		Code: 'd',
	})
	if cmd != nil {
		t.Fatalf("expected no command for direction toggle")
	}

	updated, ok := updatedModel.(*tuiModel)
	if !ok {
		t.Fatalf("expected *tuiModel, got %T", updatedModel)
	}
	if updated.directionMode != directionModeManual {
		t.Fatalf("expected manual direction mode, got %v", updated.directionMode)
	}
	if updated.converterDirection() != directionForeignToPLN {
		t.Fatalf("expected directionForeignToPLN after toggle, got %v", updated.converterDirection())
	}
}

func TestUpdateConverterComponentsSwitchesBackToAutoDirection(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.focus = 1
	model.directionMode = directionModeManual
	model.direction = directionForeignToPLN
	_ = model.plnAmountInput.Focus()

	updatedModel, _ := model.updateConverterComponents(tea.KeyPressMsg{
		Text: "1",
		Code: '1',
	})

	updated, ok := updatedModel.(*tuiModel)
	if !ok {
		t.Fatalf("expected *tuiModel, got %T", updatedModel)
	}
	if updated.directionMode != directionModeAuto {
		t.Fatalf("expected direction mode to return to auto, got %v", updated.directionMode)
	}
	if updated.converterDirection() != directionPLNToForeign {
		t.Fatalf("expected auto direction PLN->foreign after PLN edit, got %v", updated.converterDirection())
	}
}

func TestHandleConverterKeyDownMovesFocusToNextField(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.focus = 1

	updatedModel, _ := model.handleConverterKey(tea.KeyPressMsg{Code: tea.KeyDown})

	updated, ok := updatedModel.(*tuiModel)
	if !ok {
		t.Fatalf("expected *tuiModel, got %T", updatedModel)
	}
	if updated.focus != 2 {
		t.Fatalf("expected down arrow to move focus to next field, got focus %d", updated.focus)
	}
}

func TestSplitPaneWidthsStacksWhenAreaIsTooSmall(t *testing.T) {
	t.Parallel()

	left, right, stacked := splitPaneWidths(50, 28, 32)
	if !stacked {
		t.Fatalf("expected stacked layout for narrow width")
	}
	if left != 50 || right != 50 {
		t.Fatalf("expected stacked panes to reuse the full width, got left=%d right=%d", left, right)
	}
}

func TestRenderCurrencyPickerModalStaysInsideBodyArea(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	modal := model.renderCurrencyPickerModal(30, 10)

	if width := lipgloss.Width(modal); width > 30 {
		t.Fatalf("expected modal width <= 30, got %d", width)
	}
	if height := lipgloss.Height(modal); height > 10 {
		t.Fatalf("expected modal height <= 10, got %d", height)
	}
}

func TestRenderHeaderIncludesDividerBetweenTitleAndTabs(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	header := model.renderHeader(60)

	if !strings.Contains(header, strings.Repeat("─", 10)) {
		t.Fatalf("expected header divider to be rendered, got %q", header)
	}
}

func TestPrioritizeCurrenciesMovesCommonCodesToTop(t *testing.T) {
	t.Parallel()

	currencies := []models.Currency{
		{Code: "NOK", Name: "korona norweska"},
		{Code: "GBP", Name: "funt szterling"},
		{Code: "CZK", Name: "korona czeska"},
		{Code: "CHF", Name: "frank szwajcarski"},
		{Code: "USD", Name: "dolar amerykański"},
		{Code: "EUR", Name: "euro"},
	}

	prioritized := prioritizeCurrencies(currencies)
	got := []string{
		prioritized[0].Code,
		prioritized[1].Code,
		prioritized[2].Code,
		prioritized[3].Code,
	}
	want := []string{"USD", "EUR", "CHF", "GBP"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected common currencies on top in order %v, got %v", want, got)
		}
	}
}

func TestResetCurrencyFilterToSelectionRestoresFullList(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.currencies = []models.Currency{
		{Code: "USD", Name: "dolar amerykański"},
		{Code: "EUR", Name: "euro"},
		{Code: "CHF", Name: "frank szwajcarski"},
	}
	_ = model.currencyList.SetItems([]list.Item{
		converterCurrencyItem{code: "USD", name: "dolar amerykański"},
		converterCurrencyItem{code: "EUR", name: "euro"},
		converterCurrencyItem{code: "CHF", name: "frank szwajcarski"},
	})
	model.currencyList.Select(1)
	model.currencyList.SetFilterText("eur")
	model.currencyList.SetFilterState(list.FilterApplied)

	model.resetCurrencyFilterToSelection()

	if model.currencyList.FilterState() != list.Unfiltered {
		t.Fatalf("expected converter list to return to unfiltered state, got %v", model.currencyList.FilterState())
	}
	selected, ok := model.currencyList.SelectedItem().(converterCurrencyItem)
	if !ok {
		t.Fatalf("expected selected converter currency item")
	}
	if selected.code != "EUR" {
		t.Fatalf("expected EUR to stay selected after filter reset, got %q", selected.code)
	}
}

func TestSelectedCacheCurrencyCodesReturnsAllCurrencies(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.currencies = []models.Currency{
		{Code: "USD", Name: "dolar amerykański"},
		{Code: "EUR", Name: "euro"},
		{Code: "GBP", Name: "funt szterling"},
	}

	codes := model.selectedCacheCurrencyCodes()
	if len(codes) != 3 {
		t.Fatalf("expected all currencies to be returned, got %v", codes)
	}
	if codes[0] != "USD" || codes[1] != "EUR" || codes[2] != "GBP" {
		t.Fatalf("expected codes in model order, got %v", codes)
	}
}

func TestResizeExpandsConverterCurrencyListToPanelHeight(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.width = 140
	model.height = 40

	model.resize()

	expectedHeight := maxInt(model.bodyAreaHeight()-4, 6)
	if model.currencyList.Height() != expectedHeight {
		t.Fatalf("expected currency list height %d, got %d", expectedHeight, model.currencyList.Height())
	}
}

func TestRenderDatabaseBodyFillsAvailableHeight(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.width = 120
	model.height = 40
	model.resize()

	bodyHeight := model.bodyAreaHeight()
	body := model.renderDatabaseBody(100, bodyHeight)

	if lipgloss.Height(body) != bodyHeight {
		t.Fatalf("expected database body height %d, got %d", bodyHeight, lipgloss.Height(body))
	}
}

func TestHandleKeyCtrlTabSwitchesTabFromFocusedInput(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.activeTab = 0
	model.focus = 1

	updatedModel, _ := model.handleKey(tea.KeyPressMsg{
		Code: tea.KeyTab,
		Mod:  tea.ModCtrl,
	})

	updated, ok := updatedModel.(*tuiModel)
	if !ok {
		t.Fatalf("expected *tuiModel, got %T", updatedModel)
	}
	if updated.activeTab != 1 {
		t.Fatalf("expected ctrl+tab to switch to next tab, got tab %d", updated.activeTab)
	}
}

func TestHandleKeyRightArrowDoesNotSwitchTabs(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.activeTab = 0
	model.focus = 4

	updatedModel, _ := model.handleKey(tea.KeyPressMsg{Code: tea.KeyRight})

	updated, ok := updatedModel.(*tuiModel)
	if !ok {
		t.Fatalf("expected *tuiModel, got %T", updatedModel)
	}
	if updated.activeTab != 0 {
		t.Fatalf("expected right arrow not to switch tabs, got tab %d", updated.activeTab)
	}
}

func TestOnMouseClicksCacheImportButton(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.width = 120
	model.height = 40
	model.activeTab = 1
	model.resize()

	bodyX := 3
	bodyY := 2 + lipgloss.Height(model.renderHeader(maxInt(model.width-4, 10))) + 1
	formContentY := bodyY + 2
	bodyWidth := maxInt(model.shellInnerWidth()-2, 40)
	cacheFormWidth, _, _ := splitPaneWidths(bodyWidth, 34, 32)

	cmd := model.onMouse()(tea.MouseClickMsg(tea.Mouse{
		X:      bodyX + minInt(maxInt(cacheFormWidth-4, 18)-1, 10),
		Y:      formContentY + 12,
		Button: tea.MouseLeft,
	}))
	if cmd == nil {
		t.Fatalf("expected mouse command for cache import button click")
	}

	msg := cmd()
	mouseMsg, ok := msg.(mouseActionMsg)
	if !ok {
		t.Fatalf("expected mouseActionMsg, got %T", msg)
	}
	if mouseMsg.action != "prefetch-cache" {
		t.Fatalf("expected prefetch-cache action, got %q", mouseMsg.action)
	}
}

func TestOnMouseClicksDatabaseFilterField(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.width = 120
	model.height = 40
	model.activeTab = 2
	model.resize()

	bodyX := 3
	bodyY := 2 + lipgloss.Height(model.renderHeader(maxInt(model.width-4, 10))) + 1

	cmd := model.onMouse()(tea.MouseClickMsg(tea.Mouse{
		X:      bodyX + 3,
		Y:      bodyY + 3,
		Button: tea.MouseLeft,
	}))
	if cmd == nil {
		t.Fatalf("expected mouse command for database filter click")
	}

	msg := cmd()
	mouseMsg, ok := msg.(mouseActionMsg)
	if !ok {
		t.Fatalf("expected mouseActionMsg, got %T", msg)
	}
	if mouseMsg.action != "focus-db-filter" {
		t.Fatalf("expected focus-db-filter action, got %q", mouseMsg.action)
	}
}

func TestOnMouseClicksDatabaseListUsesTwoLineItems(t *testing.T) {
	t.Parallel()

	model := newTUIModel("", models.DefaultConfig(), nil, nil)
	model.width = 120
	model.height = 40
	model.activeTab = 2
	model.resize()
	_ = model.dbCurrencyList.SetItems([]list.Item{
		dbCurrencyItem{stat: cache.CurrencyStat{Code: "USD", Name: "dolar", RateCount: 10}},
		dbCurrencyItem{stat: cache.CurrencyStat{Code: "EUR", Name: "euro", RateCount: 12}},
		dbCurrencyItem{stat: cache.CurrencyStat{Code: "GBP", Name: "funt", RateCount: 8}},
	})

	bodyX := 3
	bodyY := 2 + lipgloss.Height(model.renderHeader(maxInt(model.width-4, 10))) + 1
	bodyWidth := maxInt(model.shellInnerWidth()-2, 40)
	leftWidth, _, _ := splitPaneWidths(bodyWidth, 28, 32)
	firstItemY := bodyY + 14

	cmd := model.onMouse()(tea.MouseClickMsg(tea.Mouse{
		X:      bodyX + minInt(leftWidth-3, 10),
		Y:      firstItemY + 2,
		Button: tea.MouseLeft,
	}))
	if cmd == nil {
		t.Fatalf("expected mouse command for database list click")
	}

	msg := cmd()
	clicked, ok := msg.(currencyClickedMsg)
	if !ok {
		t.Fatalf("expected currencyClickedMsg, got %T", msg)
	}
	if clicked.target != "db" || clicked.index != 1 {
		t.Fatalf("expected click to target second database item, got target=%q index=%d", clicked.target, clicked.index)
	}
}
