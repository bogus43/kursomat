# Kursomat

Kursomat to wieloplatformowa aplikacja desktopowa do przeliczania walut według średnich kursów tabeli A Narodowego Banku Polskiego. Interfejs działa w Wails v2 z Reactem, a pobrane notowania są zapisywane lokalnie w SQLite.

## Funkcje

- automatyczne przeliczanie waluty na PLN i PLN na walutę dla wybranej daty,
- przypinanie ulubionych walut oraz kopiowanie wyniku do schowka,
- automatyczne użycie ostatniego dostępnego notowania dla dnia wolnego,
- zbiorcze pobieranie wielu walut i zakresów dat,
- bieżący postęp oraz anulowanie długiego importu,
- przegląd walut, zakresu danych, wykresu trendu i 120 ostatnich notowań w bazie,
- eksport pełnej historii wybranej waluty do CSV albo JSON,
- filtrowanie, sortowanie i bezpieczne czyszczenie lokalnego cache,
- edycja parametrów sieciowych i wybór pliku bazy w natywnym oknie systemowym,
- jasny i ciemny motyw zapamiętywany między uruchomieniami.

Główne widoki mieszczą się w oknie aplikacji. Przewijanie jest ograniczone do list walut i historii, które mogą zawierać wiele rekordów.

## Wymagania

- Go 1.25.8 lub nowszy zgodny z `go.mod`,
- Node.js z npm,
- Wails CLI 2.13.0,
- zależności systemowe Wails właściwe dla Windows, macOS albo Linux.

Instalacja używanej wersji Wails CLI:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
```

Instalacja zależności frontendu:

```powershell
cd frontend
npm install
cd ..
```

## Uruchamianie

Tryb programistyczny z przeładowywaniem frontendu:

```powershell
wails dev
```

Na Windows można użyć skrótu:

```powershell
.\run.bat
```

## Budowanie

Pakiet dla aktualnego systemu operacyjnego buduje się poleceniem:

```powershell
wails build -clean
```

Na Windows dostępny jest także:

```powershell
.\build.bat
```

Wynik trafia do `build/bin/`. Wails buduje natywny pakiet na systemie, na którym jest uruchomiony, dlatego wydania dla Windows, macOS i Linux należy przygotować na odpowiednich systemach lub runnerach CI.

## Obsługa

1. W widoku **Konwerter** wybierz kierunek, walutę, datę i kwotę. Wynik jest aktualizowany automatycznie po krótkiej przerwie w edycji; automat można wyłączyć i używać przycisku **Przelicz kwotę**.
2. W widoku **Dane NBP** ustaw zakres dat, zaznacz waluty i rozpocznij import. Operację można anulować bez zamykania aplikacji.
3. W widoku **Baza** filtruj zapisane waluty, wybierz pozycję, przejrzyj trend i historię albo wyeksportuj wszystkie notowania do CSV/JSON.
4. Przycisk ustawień w panelu bocznym otwiera konfigurację bazy, timeoutu, ponowień i diagnostyki.
5. Przycisk słońca lub księżyca przełącza jasny i ciemny motyw.

## Dane i konfiguracja

Domyślne lokalizacje są wyznaczane przez system:

- konfiguracja: katalog konfiguracji użytkownika, podkatalog `kursomat/kursomat.json`,
- baza SQLite: katalog cache użytkownika, podkatalog `kursomat/kursomat.db`,
- log diagnostyczny: obok pliku bazy, gdy diagnostyka jest włączona.

Przy pierwszym uruchomieniu aplikacja rozpoznaje starsze pliki `config/kursomat.json` oraz `config/kursownik-nbp.json` i migruje ustawienia do nowej lokalizacji. Ścieżkę bazy można później zmienić w ustawieniach.

Konfigurację wdrożeniową można nadpisać zmiennymi środowiskowymi:

- `KURSOMAT_CACHE_PATH`,
- `KURSOMAT_TIMEOUT_SECONDS`,
- `KURSOMAT_RETRY_COUNT`,
- `KURSOMAT_MAX_LOOKBACK_DAYS`,
- `KURSOMAT_VERBOSE`.

## Weryfikacja

```powershell
go test ./...
go vet ./...
cd frontend
npm run build
```

Pełne sprawdzenie integracji desktopowej wykonuje `wails build -clean`.

## Struktura

```text
main.go                         # start natywnej aplikacji Wails
app.go                          # adapter metod dostępnych dla interfejsu
internal/appcore                # przypadki użycia, konfiguracja i walidacja
internal/nbp                    # klient API NBP, retry i obsługa dat
internal/cache                  # repozytorium SQLite
internal/models                 # wspólne modele domenowe
frontend/src                    # React, widoki i motywy
build                           # ikony, metadane i wynik budowania
```

Projekt nie udostępnia już interfejsu CLI ani TUI.
