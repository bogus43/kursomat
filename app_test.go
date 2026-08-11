package main

import (
	"context"
	"testing"
)

func TestRequireIdleImport(t *testing.T) {
	t.Parallel()

	app := NewDesktopApp()
	if err := app.requireIdleImport("zmienić ustawień"); err != nil {
		t.Fatalf("requireIdleImport() idle error = %v", err)
	}

	app.importCancel = context.CancelFunc(func() {})
	if err := app.requireIdleImport("zmienić ustawień"); err == nil {
		t.Fatal("requireIdleImport() expected an active import error")
	}
}
