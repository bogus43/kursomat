package main

import (
	"context"
	"errors"
	"testing"
)

func TestCancelImport(t *testing.T) {
	app := NewDesktopApp()
	if app.CancelImport() {
		t.Fatal("idle import reported cancellation")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.importCancel = cancel
	if !app.CancelImport() || !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("active import context was not canceled")
	}
	if !app.CancelImport() {
		t.Fatal("repeated cancellation should be safe until import cleanup")
	}
	app.importCancel = nil
	if app.CancelImport() {
		t.Fatal("completed import reported cancellation")
	}
}
