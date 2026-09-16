# Weryfikacja i poprawki — 2026-09-16

Naprawiono wykryte błędy wyboru kursu, walidacji, parsowania i obliczeń. Próby audytowe w `internal/appcore/audit_test.go` i `internal/nbp/audit_test.go` są teraz częścią standardowego `go test ./...`, bez specjalnego znacznika kompilacji.

## Poprawione zachowania

| Problem | Poprawka | Dowód regresyjny |
| --- | --- | --- |
| Starszy rekord cache blokował aktualniejszy kurs NBP | Tylko dokładna data lub zweryfikowane przypisanie pozwala ominąć API | `TestAuditPartialCacheMustNotHideNewerNBPRate` |
| Import nowszego notowania nie zmieniał starego przypisania | Odczyt przypisania sprawdza, czy istnieje nowszy kurs w dozwolonym zakresie | `TestAuditHistoricalImportMustRefreshResolvedQuery` |
| Stare lub przedpublikacyjne przypisania pozostawały wiarygodne | Addytywna kolumna `verified`; stare wpisy zachowane, ale ponownie weryfikowane; dzisiejszy fallback nie jest utrwalany jako zweryfikowany | `TestLegacyQueryMappingsRequireRevalidation`, `TestProvisionalMappingCannotBecomePermanent`, `TestTodaysFallbackIsRefreshed` |
| Kurs innej waluty był przyjmowany pod żądanym kodem | Kontrola kodu i tabeli odpowiedzi | `TestAuditRejectMismatchedResponseCurrency` |
| Niepoprawne daty, zerowe i ujemne kursy trafiały do SQLite | Walidacja całej odpowiedzi i całej partii przed zapisem | `TestAuditImportRejectsInvalidRowsBeforePersistence`, `TestInvalidBatchCannotPartiallyPersist` |
| Wyszukiwanie przekraczało limit zakresu API | Okna maksymalnie 93 dni, bez luk i nakładania | `TestAuditLookbackRequestFitsNBPWindow`, `TestLookbackContinuesAcrossEmptyWindows` |
| Wynik mógł być nieskończony lub źle zaokrąglony | Arytmetyka dziesiętna `big.Rat`, jedno zaokrąglenie do 4 miejsc, kontrola zakresu wyniku | `TestAuditConversionResultRemainsFinite`, `TestAuditKnownConversions` |
| Pusty tekst dawał zero, `0x10` dawało 16 | Jawny parser kwot zamiast dowolnej konwersji JavaScript | `frontend/tests/conversion.test.mjs` |
| Wynik lub błąd pozostawał po zmianie danych | Unieważnianie żądań i wyników, usunięcie starego błędu po sukcesie | `frontend/tests/browser-smoke.cjs` |

Poprawiono także daty formularzy, aby używały lokalnego dnia, a nie obciętego znacznika UTC. Backend sprawdza daty według kalendarza NBP w Warszawie. Przeliczenia zachowują istniejący kontrakt JSON (`float64`/JavaScript `number`); nie jest to interfejs do dowolnej precyzji liczb wejściowych.

## Wykonane kontrole

| Kontrola | Wynik |
| --- | --- |
| `go test -race -count=1 -cover ./...` | PASS; appcore 69,6%, cache 61,3%, nbp 54,4% pokrycia instrukcji |
| `go vet ./...` | PASS |
| `cd frontend && npm test` | 19/19 PASS |
| `cd frontend && npm run build` | PASS — TypeScript i Vite |
| `frontend/tests/browser-smoke.cjs` w Chromium | PASS — oba kierunki, pusta i błędna kwota, spóźniona odpowiedź, błąd API, tryb automatyczny, brak błędów konsoli |
| `TestConversionThroughNBPAndSQLite` | PASS — kontrakt JSON → appcore → klient NBP z odpowiedzią testową → SQLite → obliczenie → JSON |
| `KURSOMAT_LIVE_NBP=1 go test ./internal/appcore -run TestLiveNBPConversion -v -count=1` | PASS z rzeczywistym API; pierwsze przeliczenie z API, następne z cache |
| Produkcyjny build Linux z `production,webkit2_41` | PASS — plik `build/bin/kursomat` |

Test przeglądarkowy podstawia most Wails i sprawdza rzeczywiste komponenty React; nie zastępuje testu natywnego WebView. Nie udało się potwierdzić uruchomienia natywnego okna w bezekranowym środowisku wykonawczym: tymczasowy Xvfb nie uruchomił kompilatora klawiatury. Testy logiki backendu, przeglądarki i kompilacja produkcyjna zakończyły się poprawnie.

### Dane kontrolne

Zapisany w repozytorium rekord USD z 2026-04-14 zgadza się z [odpowiedzią API NBP](https://api.nbp.pl/api/exchangerates/rates/A/USD/2026-04-14/?format=json): `mid=3.6015`, tabela `071/A/NBP/2026`.

- 100 PLN → **27,7662 USD**.
- 100 USD → **360,1500 PLN**.
- 1,00105 × 1 → **1,0011** po zaokrągleniu do czterech miejsc.

Dołączona baza SQLite przeszła `integrity_check`; zawiera 1 kurs, 1 przypisanie i 0 rekordów słownika walut. Kontrola dat, wartości i osieroconych przypisań nie wykazała problemu. Baza była otwierana wyłącznie do odczytu; przed i po pracy SHA-256 wynosił `69c3be0b6585bb2daa168107273372ffad3f7c6199839636d274f59599e5cf0e`. Testy migracji i zapisów używają baz tymczasowych.

Limit 93 dni, formaty danych i początek archiwum sprawdzono w [dokumentacji NBP](https://api.nbp.pl/). Wyliczenia przykładów sprawdzono również niezależnie za pomocą arytmetyki dziesiętnej.

## Uruchomienie i powtarzanie testów

Na graficznym systemie Linux z GTK3 i WebKitGTK 4.1 uruchom `./build/bin/kursomat`. Pakiet wymaga bibliotek systemowych; zależności użyte do kompilacji w tej sesji były rozpakowane do `/tmp`, a nie instalowane w systemie. Nie jest to samowystarczalny AppImage.

Na Ubuntu zależności runtime to `libgtk-3-0t64` i `libwebkit2gtk-4.1-0`. Budowanie standardowo przez Wails opisuje [README](../README.md) i [dokumentacja Wails](https://wails.io/docs/gettingstarted/building). Samo `go build` bez znacznika `production` nie jest dowodem zbudowania działającego okna Wails.

Próba przeglądarkowa jest opcjonalna i wymaga Playwright/Chromium oraz działającego `npm run dev`. Uruchomienie: `NODE_PATH=/ścieżka/do/node_modules node frontend/tests/browser-smoke.cjs`. Zmienna `KURSOMAT_TEST_URL` pozwala wskazać inny lokalny adres; `KURSOMAT_SCREENSHOT` zapisuje zrzut. Narzędzia przeglądarkowe nie zostały dodane do zależności aplikacji.

Mapa architektury: [REPOSITORY_MAP.md](REPOSITORY_MAP.md). Reguły użycia context-mode, graphify i Sereny: [AGENTS.md](../AGENTS.md).
