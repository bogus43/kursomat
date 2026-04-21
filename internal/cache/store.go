package cache

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"kursomat/internal/models"
)

var ErrCorruptedCache = errors.New("uszkodzony cache")

type Info struct {
	Path          string
	Entries       int
	QueryMappings int
	CurrencyCount int
	LastSavedAt   string
	SizeBytes     int64
}

type CurrencyStat struct {
	Code      string
	Name      string
	RateCount int
	FirstDate string
	LastDate  string
}

type CurrencyHistoryEntry struct {
	EffectiveRateDate string
	Mid               float64
	TableNo           string
}

type Store interface {
	GetByQuery(currency, requestedDate string) (models.RateResult, bool, error)
	GetLatestRate(currency, requestedDate string) (models.RateResult, bool, error)
	StoreResolvedRate(currency, requestedDate string, rate models.NBPRate) error
	StoreHistoricalRates(currency string, rates []models.NBPRate) error
	GetCurrencies() ([]models.Currency, bool, error)
	ListCurrencyStats() ([]CurrencyStat, error)
	ListCurrencyHistory(currency string, limit int) ([]CurrencyHistoryEntry, error)
	StoreCurrencies(currencies []models.Currency) error
	Save() error
	Clear() error
	Info() (Info, error)
	Close() error
}

type sqliteStore struct {
	path string
	db   *sql.DB
}

func NewFileStore(path string) (Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("nie udało się utworzyć katalogu danych: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, normalizeDBError(fmt.Errorf("nie udało się otworzyć bazy cache: %w", err))
	}

	store := &sqliteStore{
		path: path,
		db:   db,
	}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *sqliteStore) GetByQuery(currency, requestedDate string) (models.RateResult, bool, error) {
	const query = `
SELECT r.currency_code, qc.requested_date, r.effective_rate_date, r.mid, COALESCE(r.table_no, '')
FROM query_cache qc
JOIN rates r
  ON r.currency_code = qc.currency_code
 AND r.effective_rate_date = qc.effective_rate_date
WHERE qc.currency_code = ? AND qc.requested_date = ?`

	var result models.RateResult
	err := s.db.QueryRow(query, strings.ToUpper(currency), requestedDate).Scan(
		&result.Currency,
		&result.RequestedDate,
		&result.EffectiveRateDate,
		&result.Mid,
		&result.TableNo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.RateResult{}, false, nil
	}
	if err != nil {
		return models.RateResult{}, false, normalizeDBError(fmt.Errorf("nie udało się odczytać kursu z bazy cache: %w", err))
	}
	result.Source = "cache"
	return result, true, nil
}

func (s *sqliteStore) GetLatestRate(currency, requestedDate string) (models.RateResult, bool, error) {
	const query = `
SELECT currency_code, effective_rate_date, mid, COALESCE(table_no, '')
FROM rates
WHERE currency_code = ? AND effective_rate_date <= ?
ORDER BY effective_rate_date DESC
LIMIT 1`

	var result models.RateResult
	err := s.db.QueryRow(query, strings.ToUpper(currency), requestedDate).Scan(
		&result.Currency,
		&result.EffectiveRateDate,
		&result.Mid,
		&result.TableNo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.RateResult{}, false, nil
	}
	if err != nil {
		return models.RateResult{}, false, normalizeDBError(fmt.Errorf("nie udało się odczytać historycznego kursu z bazy cache: %w", err))
	}

	result.RequestedDate = requestedDate
	result.Source = "cache"
	return result, true, nil
}

func (s *sqliteStore) StoreResolvedRate(currency, requestedDate string, rate models.NBPRate) error {
	tx, err := s.db.Begin()
	if err != nil {
		return normalizeDBError(fmt.Errorf("nie udało się rozpocząć zapisu do bazy cache: %w", err))
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	timestamp := time.Now().Format(time.RFC3339)
	currency = strings.ToUpper(currency)

	if _, err = tx.Exec(`
INSERT INTO rates(currency_code, effective_rate_date, mid, table_no, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(currency_code, effective_rate_date) DO UPDATE SET
  mid = excluded.mid,
  table_no = excluded.table_no,
  updated_at = excluded.updated_at`,
		currency,
		rate.EffectiveRateDate,
		rate.Mid,
		rate.TableNo,
		timestamp,
	); err != nil {
		return normalizeDBError(fmt.Errorf("nie udało się zapisać kursu do bazy cache: %w", err))
	}

	if _, err = tx.Exec(`
INSERT INTO query_cache(currency_code, requested_date, effective_rate_date, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(currency_code, requested_date) DO UPDATE SET
  effective_rate_date = excluded.effective_rate_date,
  updated_at = excluded.updated_at`,
		currency,
		requestedDate,
		rate.EffectiveRateDate,
		timestamp,
	); err != nil {
		return normalizeDBError(fmt.Errorf("nie udało się zapisać mapowania zapytania do bazy cache: %w", err))
	}

	if err = tx.Commit(); err != nil {
		return normalizeDBError(fmt.Errorf("nie udało się zatwierdzić zapisu do bazy cache: %w", err))
	}
	return nil
}

func (s *sqliteStore) StoreHistoricalRates(currency string, rates []models.NBPRate) error {
	if len(rates) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return normalizeDBError(fmt.Errorf("nie udało się rozpocząć zapisu zakresu kursów do bazy cache: %w", err))
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	timestamp := time.Now().Format(time.RFC3339)
	currency = strings.ToUpper(currency)
	for _, rate := range rates {
		if _, err = tx.Exec(`
INSERT INTO rates(currency_code, effective_rate_date, mid, table_no, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(currency_code, effective_rate_date) DO UPDATE SET
  mid = excluded.mid,
  table_no = excluded.table_no,
  updated_at = excluded.updated_at`,
			currency,
			rate.EffectiveRateDate,
			rate.Mid,
			rate.TableNo,
			timestamp,
		); err != nil {
			return normalizeDBError(fmt.Errorf("nie udało się zapisać kursu historycznego do bazy cache: %w", err))
		}
	}

	if err = tx.Commit(); err != nil {
		return normalizeDBError(fmt.Errorf("nie udało się zatwierdzić zapisu zakresu kursów do bazy cache: %w", err))
	}
	return nil
}

func (s *sqliteStore) GetCurrencies() ([]models.Currency, bool, error) {
	rows, err := s.db.Query(`SELECT code, name FROM currencies ORDER BY code`)
	if err != nil {
		return nil, false, normalizeDBError(fmt.Errorf("nie udało się odczytać listy walut z bazy cache: %w", err))
	}
	defer rows.Close()

	currencies := make([]models.Currency, 0, 32)
	for rows.Next() {
		var currency models.Currency
		if err := rows.Scan(&currency.Code, &currency.Name); err != nil {
			return nil, false, normalizeDBError(fmt.Errorf("nie udało się odczytać rekordu waluty z bazy cache: %w", err))
		}
		currencies = append(currencies, currency)
	}
	if err := rows.Err(); err != nil {
		return nil, false, normalizeDBError(fmt.Errorf("nie udało się zakończyć odczytu listy walut z bazy cache: %w", err))
	}
	return currencies, len(currencies) > 0, nil
}

func (s *sqliteStore) ListCurrencyStats() ([]CurrencyStat, error) {
	const query = `
WITH known_currencies AS (
  SELECT code, name FROM currencies
  UNION
  SELECT DISTINCT currency_code AS code, currency_code AS name
  FROM rates
  WHERE currency_code NOT IN (SELECT code FROM currencies)
)
SELECT
  kc.code,
  kc.name,
  COUNT(r.effective_rate_date) AS rate_count,
  COALESCE(MIN(r.effective_rate_date), '') AS first_date,
  COALESCE(MAX(r.effective_rate_date), '') AS last_date
FROM known_currencies kc
LEFT JOIN rates r ON r.currency_code = kc.code
GROUP BY kc.code, kc.name
ORDER BY kc.code`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, normalizeDBError(fmt.Errorf("nie udało się odczytać statystyk walut z bazy cache: %w", err))
	}
	defer rows.Close()

	stats := make([]CurrencyStat, 0, 16)
	for rows.Next() {
		var stat CurrencyStat
		if err := rows.Scan(&stat.Code, &stat.Name, &stat.RateCount, &stat.FirstDate, &stat.LastDate); err != nil {
			return nil, normalizeDBError(fmt.Errorf("nie udało się odczytać rekordu statystyk waluty: %w", err))
		}
		stats = append(stats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeDBError(fmt.Errorf("nie udało się zakończyć odczytu statystyk walut z bazy cache: %w", err))
	}
	return stats, nil
}

func (s *sqliteStore) ListCurrencyHistory(currency string, limit int) ([]CurrencyHistoryEntry, error) {
	query := `
SELECT effective_rate_date, mid, COALESCE(table_no, '')
FROM rates
WHERE currency_code = ?
ORDER BY effective_rate_date DESC`

	args := []any{strings.ToUpper(currency)}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, normalizeDBError(fmt.Errorf("nie udało się odczytać historii waluty z bazy cache: %w", err))
	}
	defer rows.Close()

	history := make([]CurrencyHistoryEntry, 0, 64)
	for rows.Next() {
		var entry CurrencyHistoryEntry
		if err := rows.Scan(&entry.EffectiveRateDate, &entry.Mid, &entry.TableNo); err != nil {
			return nil, normalizeDBError(fmt.Errorf("nie udało się odczytać rekordu historii waluty: %w", err))
		}
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeDBError(fmt.Errorf("nie udało się zakończyć odczytu historii waluty z bazy cache: %w", err))
	}
	return history, nil
}

func (s *sqliteStore) StoreCurrencies(currencies []models.Currency) error {
	tx, err := s.db.Begin()
	if err != nil {
		return normalizeDBError(fmt.Errorf("nie udało się rozpocząć zapisu listy walut do bazy cache: %w", err))
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	timestamp := time.Now().Format(time.RFC3339)
	for _, currency := range currencies {
		if _, err = tx.Exec(`
INSERT INTO currencies(code, name, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(code) DO UPDATE SET
  name = excluded.name,
  updated_at = excluded.updated_at`,
			strings.ToUpper(currency.Code),
			currency.Name,
			timestamp,
		); err != nil {
			return normalizeDBError(fmt.Errorf("nie udało się zapisać waluty %s do bazy cache: %w", currency.Code, err))
		}
	}

	if err = tx.Commit(); err != nil {
		return normalizeDBError(fmt.Errorf("nie udało się zatwierdzić zapisu listy walut do bazy cache: %w", err))
	}
	return nil
}

func (s *sqliteStore) Save() error {
	return nil
}

func (s *sqliteStore) Clear() error {
	if _, err := s.db.Exec(`DELETE FROM query_cache; DELETE FROM rates; DELETE FROM currencies;`); err != nil {
		return normalizeDBError(fmt.Errorf("nie udało się wyczyścić bazy cache: %w", err))
	}
	return nil
}

func (s *sqliteStore) Info() (Info, error) {
	info := Info{Path: s.path}

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM rates`).Scan(&info.Entries); err != nil {
		return Info{}, normalizeDBError(fmt.Errorf("nie udało się odczytać liczby kursów z bazy cache: %w", err))
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM query_cache`).Scan(&info.QueryMappings); err != nil {
		return Info{}, normalizeDBError(fmt.Errorf("nie udało się odczytać liczby mapowań z bazy cache: %w", err))
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM currencies`).Scan(&info.CurrencyCount); err != nil {
		return Info{}, normalizeDBError(fmt.Errorf("nie udało się odczytać liczby walut z bazy cache: %w", err))
	}
	if err := s.db.QueryRow(`
SELECT COALESCE(MAX(ts), '')
FROM (
  SELECT MAX(updated_at) AS ts FROM rates
  UNION ALL
  SELECT MAX(updated_at) AS ts FROM query_cache
  UNION ALL
  SELECT MAX(updated_at) AS ts FROM currencies
)`).Scan(&info.LastSavedAt); err != nil {
		return Info{}, normalizeDBError(fmt.Errorf("nie udało się odczytać czasu ostatniego zapisu z bazy cache: %w", err))
	}

	stat, err := os.Stat(s.path)
	if err == nil {
		info.SizeBytes = stat.Size()
	}
	return info, nil
}

func (s *sqliteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *sqliteStore) initSchema() error {
	const schema = `
PRAGMA busy_timeout = 5000;
PRAGMA journal_mode = WAL;
CREATE TABLE IF NOT EXISTS rates (
  currency_code TEXT NOT NULL,
  effective_rate_date TEXT NOT NULL,
  mid REAL NOT NULL,
  table_no TEXT,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(currency_code, effective_rate_date)
);
CREATE TABLE IF NOT EXISTS query_cache (
  currency_code TEXT NOT NULL,
  requested_date TEXT NOT NULL,
  effective_rate_date TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(currency_code, requested_date)
);
CREATE TABLE IF NOT EXISTS currencies (
  code TEXT NOT NULL PRIMARY KEY,
  name TEXT NOT NULL,
  updated_at TEXT NOT NULL
);`

	if _, err := s.db.Exec(schema); err != nil {
		return normalizeDBError(fmt.Errorf("nie udało się przygotować struktury bazy cache: %w", err))
	}
	return nil
}

func normalizeDBError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "not a database") {
		return fmt.Errorf("%w: %v", ErrCorruptedCache, err)
	}
	return err
}
