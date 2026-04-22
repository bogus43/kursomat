package cli

import (
	"testing"

	tea "charm.land/bubbletea/v2"

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
