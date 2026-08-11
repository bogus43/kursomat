package appcore

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"kursomat/internal/cache"
)

func TestWriteHistoryExportCSV(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "eur.csv")
	entries := []cache.CurrencyHistoryEntry{{
		EffectiveRateDate: "2026-04-14", Mid: 4.2551, TableNo: "072/A/NBP/2026",
	}}
	if err := WriteHistoryExport(path, "csv", "EUR", entries); err != nil {
		t.Fatalf("WriteHistoryExport() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) < 3 || string(data[:3]) != "\xef\xbb\xbf" {
		t.Fatal("CSV export is missing the UTF-8 BOM")
	}
	reader := csv.NewReader(bytes.NewReader(data[3:]))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(records) != 2 || records[1][0] != "EUR" || records[1][2] != "4.2551" {
		t.Fatalf("unexpected CSV records: %#v", records)
	}
}

func TestWriteHistoryExportJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "usd.json")
	entries := []cache.CurrencyHistoryEntry{{
		EffectiveRateDate: "2026-04-14", Mid: 3.8123, TableNo: "072/A/NBP/2026",
	}}
	if err := WriteHistoryExport(path, "json", "USD", entries); err != nil {
		t.Fatalf("WriteHistoryExport() error = %v", err)
	}

	var payload HistoryExport
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload.Currency != "USD" || len(payload.Entries) != 1 || payload.Entries[0].Mid != 3.8123 {
		t.Fatalf("unexpected JSON payload: %#v", payload)
	}
}

func TestWriteHistoryExportRejectsUnsupportedFormat(t *testing.T) {
	t.Parallel()

	err := WriteHistoryExport(filepath.Join(t.TempDir(), "eur.xml"), "xml", "EUR", []cache.CurrencyHistoryEntry{{}})
	if err == nil {
		t.Fatal("WriteHistoryExport() expected an unsupported format error")
	}
}
