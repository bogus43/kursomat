# Kursownik NBP

`Kursownik NBP` to aplikacja CLI/TUI w Go do pobierania średnich kursów walut z oficjalnego API NBP (tabela A), z lokalnym cache w bazie SQLite.

Aplikacja oferuje:
- klasyczne komendy CLI (`rate`, `cache`),
- interfejs TUI (pelnoekranowy terminal UI) uruchamiany komenda `tui` lub domyslnie bez argumentow,
- konwerter dwukierunkowy `PLN <-> waluta z NBP`,
- picker waluty oparty o biblioteki Charmbracelet.

## Funkcje MVP

- pobieranie kursu dla pojedynczej waluty i daty,
- obsługa wielu walut dla jednej daty,
- konwerter dwukierunkowy w TUI dla dowolnej waluty z tabeli A NBP,
- automatyczny fallback do najbliższej wcześniejszej daty publikacji,
- lokalny cache w SQLite (bez ponownego pobierania tych samych danych),
- cache listy walut NBP do bazy,
- tryb wyjścia `text` i `json`,
- komendy zarządzania cache: `cache info`, `cache clear`,
- retry + timeout dla żądań HTTP,
- komunikaty błędów po polsku.

## Waluty

- CLI akceptuje poprawne kody ISO 4217, a weryfikacja danych odbywa sie przez odpowiedz API NBP.
- TUI ładuje i cache'uje listę walut z aktualnej tabeli A NBP, a wybór odbywa się z pickera.

## Wymagania

- Go 1.25+

## Instalacja

```bash
go build -o kursownik-nbp ./cmd/kursownik-nbp
```

## Użycie

### Start TUI (domyślny)

```bash
./kursownik-nbp
# lub
./kursownik-nbp tui
```

### Pobranie jednego kursu

```bash
./kursownik-nbp rate --currency USD --date 2026-04-14
```

### Pobranie wielu kursów

```bash
./kursownik-nbp rate --currency USD,EUR,CHF --date 2026-04-14
```

### Wyjście JSON

```bash
./kursownik-nbp rate --currency USD --date 2026-04-14 --output json
```

### Cache

```bash
./kursownik-nbp cache info
./kursownik-nbp cache clear
```

### Skróty TUI

- `←/→` przełączanie zakładek (`Konwerter` / `Baza cache`)
- `Tab` / `Shift+Tab` zmiana fokusu
- `Enter` akcja główna
- `d` odwrócenie kierunku konwersji
- `r` odśwież informacje o cache (w zakładce cache)
- `c` wyczyść cache (w zakładce cache)
- `?` pokazanie/skrycie pełnej pomocy
- `q` lub `Ctrl+C` wyjście

## Konfiguracja

Aplikacja ma sensowne wartości domyślne. Opcjonalnie można podać plik konfiguracyjny JSON przez `--config`.

Przy starcie aplikacja automatycznie tworzy:
- katalog `./data` oraz plik `./data/kursownik.db`,
- katalog `./config` oraz plik `./config/kursownik-nbp.json`.

Przykład:

```json
{
  "cache_path": "C:/tmp/kursownik.db",
  "timeout_seconds": 10,
  "retry_count": 2,
  "max_lookback_days": 92,
  "verbose": false
}
```

Flagi CLI mają wyższy priorytet niż plik konfiguracyjny:

- `--cache-path`
- `--timeout`
- `--retry`
- `--lookback-days`
- `--verbose`

## Przykład wyjścia tekstowego

```text
Waluta: USD
Data żądana: 2026-04-14
Data kursu: 2026-04-13
Kurs średni NBP: 3.8123
Tabela: 071/A/NBP/2026
Źródło: NBP API
```

## Przykład wyjścia JSON

```json
{
  "currency": "USD",
  "requested_date": "2026-04-14",
  "effective_rate_date": "2026-04-13",
  "mid": 3.8123,
  "table_no": "071/A/NBP/2026",
  "source": "NBP API"
}
```

## Architektura

```text
cmd/kursownik-nbp/main.go         # entrypoint CLI
internal/cli                      # parser komend, walidacja, output, config
internal/nbp                      # klient API NBP + retry/timeout + logika daty kursu
internal/cache                    # cache SQLite (kursy, mapa zapytań, lista walut)
internal/models                   # wspólne modele i konfiguracja
```

## Testy

Uruchomienie testów:

```bash
go test ./...
```

Zakres testów:

- walidacja wejścia (`internal/cli`),
- parser odpowiedzi API i logika wyboru daty (`internal/nbp`),
- cache (`internal/cache`),
- integracyjny test klienta NBP z `httptest`.
