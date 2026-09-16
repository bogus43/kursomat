# Instrukcje pracy w repozytorium

## context-mode

- Duże wyniki poleceń, logi i przegląd wielu plików zbieraj przez `ctx_batch_execute`; przekazuj pytania w `queries`.
- Do analizy, filtrowania i podsumowań używaj `ctx_execute` lub `ctx_execute_file`, wypisując tylko potrzebne wyniki.
- Wcześniej zindeksowane ustalenia odszukuj przez `ctx_search`; po wznowieniu sesji użyj `sort: "timeline"` i potwierdź stan w repozytorium.
- Zmiany plików wykonuj narzędziem edycji, nie w nietrwałym środowisku context-mode. Nie indeksuj sekretów ani lokalnych baz użytkownika.

## graphify

- Mapę zależności utrzymuj w `graphify-out/`. Jeśli istnieje `graph.json`, zacznij pytania o architekturę od zapytania do grafu z jawną ścieżką projektu.
- Przy pierwszym mapowaniu sprawdź dostępność narzędzia i zakres skanowania. Pomijaj zależności, artefakty budowania oraz dane użytkownika.
- Po zmianach sprawdzaj aktualność grafu. Zależności wywnioskowane odróżniaj od wydobytych ze źródeł; każdą usterkę potwierdź w kodzie i odpowiednim teście.
- Graf służy do nawigacji i nie zastępuje przeglądu źródeł ani testów.

## Serena

- Jeśli MCP Serena jest dostępny, aktywuj projekt wskazując katalog tego repozytorium i sprawdź instrukcje/onboarding projektu.
- Używaj nawigacji po symbolach oraz wyszukiwania referencji do ustalenia wpływu zmiany przed edycją.
- Edycje symboli stosuj tylko po sprawdzeniu zakresu, a wynik weryfikuj przez diff i testy.
- Gdy narzędzie lub obsługa języka są niedostępne, jawnie odnotuj ograniczenie i użyj `rg`, analizy źródeł oraz testów. Nie deklaruj wykonania operacji Sereną bez wyniku narzędzia.

## Weryfikacja danych i obliczeń

- Prześledź cały przepływ: formularz → adapter Wails → appcore → API NBP/cache → wynik.
- Sprawdzaj format kwot i dat, skończoność i zakres liczb, dodatniość kursów, zgodność waluty i daty odpowiedzi, kierunek przeliczenia oraz zaokrąglanie.
- Testy API uruchamiaj z lokalnymi odpowiedziami testowymi, a testy SQLite na bazach tymczasowych; nie modyfikuj danych użytkownika.
- Podstawowe kontrole: `go test ./...`, `go vet ./...`, kompilacja frontendu. Zapisuj konkretne ograniczenia kontroli i odróżniaj błędy środowiska od usterek programu.
