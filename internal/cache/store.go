package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kursomat/internal/models"
)

var ErrCorruptedCache = errors.New("uszkodzony cache")

type Info struct {
	Path          string
	Entries       int
	QueryMappings int
	LastSavedAt   string
	SizeBytes     int64
}

type Store interface {
	GetByQuery(currency, requestedDate string) (models.RateResult, bool, error)
	StoreResolvedRate(currency, requestedDate string, rate models.NBPRate) error
	Save() error
	Clear() error
	Info() (Info, error)
}

type fileStore struct {
	mu     sync.RWMutex
	saveMu sync.Mutex
	path   string
	file   cacheFile
}

type cacheFile struct {
	Version    int                `json:"version"`
	SavedAt    string             `json:"saved_at,omitempty"`
	Entries    map[string]rateRow `json:"entries"`
	QueryIndex map[string]string  `json:"query_index"`
}

type rateRow struct {
	Currency          string  `json:"currency"`
	EffectiveRateDate string  `json:"effective_rate_date"`
	Mid               float64 `json:"mid"`
	TableNo           string  `json:"table_no,omitempty"`
}

func NewFileStore(path string) (Store, error) {
	store := &fileStore{
		path: path,
		file: cacheFile{
			Version:    1,
			Entries:    map[string]rateRow{},
			QueryIndex: map[string]string{},
		},
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *fileStore) GetByQuery(currency, requestedDate string) (models.RateResult, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requestedKey := makeQueryKey(currency, requestedDate)
	entryKey, ok := s.file.QueryIndex[requestedKey]
	if !ok {
		return models.RateResult{}, false, nil
	}
	row, ok := s.file.Entries[entryKey]
	if !ok {
		return models.RateResult{}, false, nil
	}

	return models.RateResult{
		Currency:          row.Currency,
		RequestedDate:     requestedDate,
		EffectiveRateDate: row.EffectiveRateDate,
		Mid:               row.Mid,
		TableNo:           row.TableNo,
		Source:            "cache",
	}, true, nil
}

func (s *fileStore) StoreResolvedRate(currency, requestedDate string, rate models.NBPRate) error {
	effectiveKey := makeRateKey(currency, rate.EffectiveRateDate)
	queryKey := makeQueryKey(currency, requestedDate)

	s.mu.Lock()
	s.file.Entries[effectiveKey] = rateRow{
		Currency:          strings.ToUpper(currency),
		EffectiveRateDate: rate.EffectiveRateDate,
		Mid:               rate.Mid,
		TableNo:           rate.TableNo,
	}
	s.file.QueryIndex[queryKey] = effectiveKey
	s.file.SavedAt = time.Now().Format(time.RFC3339)
	s.mu.Unlock()

	return s.Save()
}

func (s *fileStore) Save() error {
	snapshot, err := s.snapshot()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("nie udało się zapisać cache: %w", err)
	}

	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	return writeDataAtomic(s.path, data)
}

func (s *fileStore) Clear() error {
	s.mu.Lock()
	s.file = cacheFile{
		Version:    1,
		SavedAt:    time.Now().Format(time.RFC3339),
		Entries:    map[string]rateRow{},
		QueryIndex: map[string]string{},
	}
	s.mu.Unlock()
	return s.Save()
}

func (s *fileStore) Info() (Info, error) {
	s.mu.RLock()
	info := Info{
		Path:          s.path,
		Entries:       len(s.file.Entries),
		QueryMappings: len(s.file.QueryIndex),
		LastSavedAt:   s.file.SavedAt,
	}
	s.mu.RUnlock()

	stat, err := os.Stat(s.path)
	if err == nil {
		info.SizeBytes = stat.Size()
	}
	return info, nil
}

func (s *fileStore) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("nie udało się utworzyć katalogu cache: %w", err)
	}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("nie udało się odczytać cache: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	var parsed cacheFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("%w: %v", ErrCorruptedCache, err)
	}
	if parsed.Entries == nil {
		parsed.Entries = map[string]rateRow{}
	}
	if parsed.QueryIndex == nil {
		parsed.QueryIndex = map[string]string{}
	}
	if parsed.Version == 0 {
		parsed.Version = 1
	}

	s.mu.Lock()
	s.file = parsed
	s.mu.Unlock()
	return nil
}

func (s *fileStore) snapshot() (cacheFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cacheFile{
		Version:    s.file.Version,
		SavedAt:    s.file.SavedAt,
		Entries:    cloneEntries(s.file.Entries),
		QueryIndex: cloneIndex(s.file.QueryIndex),
	}, nil
}

func cloneEntries(in map[string]rateRow) map[string]rateRow {
	out := make(map[string]rateRow, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneIndex(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func writeDataAtomic(path string, data []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("nie udało się zapisać pliku tymczasowego cache: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("nie udało się podmienić pliku cache: %w", err)
	}
	return nil
}

func makeRateKey(currency, effectiveDate string) string {
	return strings.ToUpper(currency) + "|" + effectiveDate
}

func makeQueryKey(currency, requestedDate string) string {
	return strings.ToUpper(currency) + "|" + requestedDate
}
