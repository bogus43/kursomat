package cli

import (
	"io"
	"testing"
)

func TestNewRootCommandExposesExpectedCommandsAndFlags(t *testing.T) {
	t.Parallel()

	app := NewApp(io.Discard, io.Discard)
	cmd := app.newRootCommand()

	if cmd.Name() != "kursownik-nbp" {
		t.Fatalf("expected root command name kursownik-nbp, got %q", cmd.Name())
	}

	subcommands := map[string]bool{
		"tui":   false,
		"rate":  false,
		"cache": false,
	}
	for _, subcommand := range cmd.Commands() {
		if _, ok := subcommands[subcommand.Name()]; ok {
			subcommands[subcommand.Name()] = true
		}
	}
	for name, found := range subcommands {
		if !found {
			t.Fatalf("expected subcommand %q to be registered", name)
		}
	}

	for _, flagName := range []string{"config", "cache-path", "timeout", "retry", "lookback-days", "verbose"} {
		if cmd.PersistentFlags().Lookup(flagName) == nil {
			t.Fatalf("expected persistent flag %q to be registered", flagName)
		}
	}
}
