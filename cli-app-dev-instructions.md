# CLI App Development Instructions

## Cel dokumentu

Ten dokument opisuje praktyczne zasady budowy nowoczesnych aplikacji konsolowych i TUI w Go, z naciskiem na:

- odporność na freeze i błędy renderowania,
- estetyczne skalowanie do rozmiaru terminala,
- poprawną obsługę zakładek, przycisków, pól wejściowych i modali,
- użycie skrótów klawiaturowych i paska pomocy na dole okna,
- dobór aktywnie rozwijanych bibliotek,
- reguły implementacyjne dla Codex i GitHub Copilot.

---

## Zalecany stos bazowy

### Główne biblioteki TUI

1. **Bubble Tea v2**
   - silnik stanu i pętli zdarzeń,
   - model `Update/View/Init`,
   - obsługa resize, klawiatury, trybu pełnoekranowego,
   - nadaje się do prostych i złożonych aplikacji TUI.

2. **Bubbles v2**
   - gotowe komponenty UI,
   - szczególnie przydatne: `textinput`, `textarea`, `list`, `viewport`, `spinner`, `timer`, `stopwatch`, `filepicker`.

3. **Lip Gloss v2**
   - style, layout, ramki, padding, kolory,
   - budowa zakładek, status barów, popupów i układów wielokolumnowych.
   - **Uwaga:** Lip Gloss v2 **usuwa `AdaptiveColor`**. Komponenty (`help`, `list`, `textarea`, `textinput`) oferują teraz `DefaultStyles(hasDarkBG bool)`, `DefaultLightStyles()` i `DefaultDarkStyles()`. W praktyce oznacza to, że wykrywanie tła i dobór stylów trzeba obsłużyć jawnie. Dla migracji z v1 dostępny jest pakiet `charm.land/lipgloss/v2/compat`.

### Biblioteki uzupełniające

4. **Huh v2**
   - formularze, wizardy, prompty,
   - dobry wybór dla wieloetapowych ekranów konfiguracji.

5. **Glamour**
   - render Markdown w terminalu,
   - pomocne do `Help`, `README`, dokumentacji operatora, changelogów.
   - **Dwie równoległe linie:**
     - `github.com/charmbracelet/glamour` v0.10.x — stabilna, szeroka kompatybilność,
     - `charm.land/glamour/v2` — nowe API, wymaga całego stosu `charm.land` v2.

6. **Wish (v1.x)**
   - uruchamianie aplikacji Bubble Tea po SSH,
   - każda sesja SSH dostaje osobny `tea.Program`,
   - resize PTY jest obsługiwany natywnie.
   - **Uwaga:** `charm.land/wish/v2` nie istnieje — stabilna wersja to `github.com/charmbracelet/wish` v1.x.

### Biblioteki dodatkowe spoza Charmbracelet

7. **go-pretty** (`table`, `progress`, `text`, `list`)
   - tabele, postęp, formatowanie tekstu, listy,
   - szczególnie dobre do raportów i batchowych pipeline’ów.

8. **Cobra**
   - standard do budowy klasycznych CLI w Go,
   - dobra baza dla aplikacji z podkomendami, flagami, shell completion i generacją helpów.

9. **Viper**
   - konfiguracja aplikacji,
   - pliki konfiguracyjne, env vars, profile runtime.

10. **PTerm**
    - szybkie budowanie ładnego CLI bez pełnego TUI,
    - oferuje tabele, progress bary, drzewa, prompty, selekty.

11. **termenv**
    - bezpieczna obsługa ANSI i kolorów,
    - detekcja możliwości terminala,
    - szczególnie przydatne dla aplikacji nieopartych bezpośrednio o Lip Gloss.

12. **tcell**
    - niskopoziomowa biblioteka terminalowa,
    - dobra, gdy potrzebna jest pełna kontrola nad komórkami ekranu,
    - sensowna alternatywa dla własnych rendererów low-level.

13. **muesli/reflow**
    - ANSI-aware zawijanie i reflow tekstu,
    - przydatne dla logów, opisów, Markdown preview i help screenów.

14. **nao1215/prompt**
    - nowocześniejsza biblioteka promptów,
    - deklarowana jako zamiennik dla nieutrzymywanego `c-bata/go-prompt`.

15. **briandowns/spinner**
    - prosty spinner dla klasycznych CLI,
    - sensowny poza pełnym TUI albo przy pomocniczych taskach terminalowych.

16. **charm.land/log v2**
    - strukturalne logowanie (logfmt/JSON) z poziomami,
    - kolorowe wyjście zintegrowane z Lip Gloss v2 (automatyczny downsampling kolorów),
    - naturalne uzupełnienie reguły „loguj do pliku" — zamiast `log.New(f, ...)` ze stdlib,
    - import: `charm.land/log/v2`.

17. **charmbracelet/colorprofile**
    - wykrywanie profilu kolorów terminala: `TrueColor`, `ANSI256`, `ANSI`, `Ascii`, `NoTTY`,
    - preferowane narzędzie w stacku v2 do jawnego wykrywania albo wymuszania profilu,
    - przydatne, gdy chcesz świadomie dobrać lub nadpisać profil kolorów w Bubble Tea v2,
    - import: `github.com/charmbracelet/colorprofile`.

---

## Nieaktualne / zarchiwizowane repozytoria

- **charmbracelet/charm** (Charm Cloud, KV, FS, Accounts) — zarchiwizowane przez właściciela w **marcu 2025**. Nie używaj `github.com/charmbracelet/charm` w nowych projektach.
- **AlecAivazis/survey** — archived, nie jest utrzymywany.
- **c-bata/go-prompt** — nieaktywny; rekomendowanym zamiennikiem jest `nao1215/prompt`.

---

## Cechy dobrej i nowoczesnej aplikacji konsolowej

### 1. Jeden centralny model stanu

Aplikacja powinna mieć jeden główny model, który przechowuje:

- aktywną zakładkę,
- rozmiar terminala,
- stan popupów i modali,
- stan ładowania,
- dane domenowe,
- błędy,
- focus aktywnych kontrolek.

**Zasada:** UI jest funkcją stanu. Nie odwrotnie.

### 2. Przewidywalna obsługa klawiatury

Dobra aplikacja TUI musi działać sensownie bez myszy.

Minimalny standard:

- `Tab` / `Shift+Tab` — zmiana fokusu,
- `←` / `→` — przełączanie zakładek,
- `↑` / `↓` — zmiana fokusu lub selekcji w aktualnym panelu,
- `Enter` — akcja główna,
- `Esc` — anulowanie / zamknięcie modala,
- `q` — wyjście albo powrót, jeśli to zgodne z kontekstem,
- `Ctrl+C` — przerwanie albo awaryjne wyjście, zależnie od typu aplikacji,
- `/` — wejście w tryb wyszukiwania albo filtrowania,
- `?` albo `F1` — ekran pomocy.

### 3. Responsywny layout

Układ musi reagować na `tea.WindowSizeMsg` i przeliczać:

- szerokości boxów,
- wysokości viewportów,
- rozmiary textarea,
- pozycję popupów,
- szerokość status bara,
- szerokość tabel i list.

### 4. Czytelna hierarchia wizualna

Każdy ekran powinien mieć co najmniej:

- górny pasek / zakładki,
- obszar roboczy,
- dolny pasek skrótów,
- spójny obszar błędów / komunikatów,
- osobny modal/popup dla akcji blokujących.

W praktyce profesjonalny TUI powinien też mieć spójny kontener wizualny:

- główny obszar aplikacji zwykle powinien być osadzony w jednej czytelnej ramce lub wyraźnie wydzielonym kontenerze,
- dodatkowe ramki wewnętrzne należy stosować oszczędnie,
- ramki mają poprawiać orientację i skalowanie przy resize, a nie zabierać przestrzeń bez wartości UX.

### 5. Wyraźne stany pośrednie

Każdy panel danych powinien obsługiwać:

- `empty state`,
- `loading state`,
- `loaded state`,
- `error state`,
- `busy/processing state`.

### 6. Ograniczanie i przewijanie treści

Żaden panel nie powinien rosnąć bez końca.

W praktyce:

- duże treści w `viewport`,
- logi w przewijanym widoku,
- preview przycinane do liczby linii,
- textarea z ograniczonym viewportem,
- długie linie zawijane lub obcinane.

### 7. Kolory muszą być odporne na środowisko

Nie zakładaj, że każdy terminal obsługuje TrueColor. Korzystaj z bibliotek, które wykrywają profil kolorów i potrafią zredukować paletę.

### 8. Debug i logowanie poza stdout

Nie używaj `fmt.Println()` do debugowania podczas aktywnego TUI. Loguj do pliku albo do osobnego loggera.

---

## Jak przeciwdziałać freeze

W aplikacjach terminalowych „freeze” najczęściej nie oznacza realnego zawieszenia procesu, tylko błąd architektury widoku, blokowanie pętli zdarzeń albo uszkodzony layout.

### Typowe źródła freeze

1. **Długie operacje wykonywane bezpośrednio w `Update()`**
   - sieć,
   - odczyt plików,
   - parsowanie dużych danych,
   - sleep,
   - kosztowne obliczenia.

2. **Błędny modal/popup**
   - doklejanie drugiego pełnego ekranu pod pierwszy,
   - renderowanie dwóch wielkich widoków zamiast jednego modala,
   - błędne nakładanie warstw.

3. **Nieograniczone dane w widoku**
   - długie logi,
   - pełne preview textarea,
   - brak `viewport`,
   - brak limitu linii.

4. **Wyścigi stanu lub zbyt agresywne odświeżanie**
   - równoczesne modyfikacje tego samego modelu,
   - błędne pętle timera,
   - nieskończone generowanie `Cmd`.

5. **Pisanie do stdout podczas renderowania TUI**
   - psuje ramki,
   - rozjeżdża ekran,
   - wygląda jak hang.

### Reguły zapobiegania freeze

#### Reguła A — nie blokuj `Update()`

`Update()` powinno tylko:

- przyjąć event,
- zmienić stan,
- zwrócić `Cmd` albo `nil`.

Nie umieszczaj w nim bezpośrednio długich operacji.

#### Reguła B — długie zadania uruchamiaj przez `tea.Cmd`

Wzorzec:

```go
func runJobCmd(input JobInput) tea.Cmd {
    return func() tea.Msg {
        result, err := RunJob(input)
        return jobFinishedMsg{result: result, err: err}
    }
}
```

#### Reguła C — modal blokuje wejście, ale nie renderuje drugiego pełnego ekranu pod spodem

Modal powinien być osobnym trybem renderowania, nie „doklejonym tekstem” do zwykłego widoku.

#### Reguła D — wprowadzaj stany `loading` / `busy`

Po starcie długiego zadania ustaw:

- `loading = true`,
- zablokuj wybrane inputy,
- pokaż spinner/progress,
- po otrzymaniu wiadomości końcowej ustaw `loading = false`.

#### Reguła E — każda duża treść musi mieć limit

- logi: max N wierszy lub viewport,
- podgląd tekstu: max N linii i max szerokość,
- markdown: viewport,
- output z procesu: incremental append do przewijanego panelu.

#### Reguła F — loguj do pliku

Przykład:

```go
f, _ := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
logger := log.New(f, "", log.LstdFlags|log.Lshortfile)
```

#### Reguła G — osobne wiadomości dla startu i końca zadania

- `startJobMsg`
- `jobProgressMsg`
- `jobFinishedMsg`
- `jobFailedMsg`

To upraszcza debug i sterowanie UI.

### Anti-freeze checklist

- żadnych `time.Sleep()` w `Update()`,
- żadnych ciężkich pętli w `View()`,
- żadnych logów do stdout,
- brak nieograniczonego renderu textarea/logów,
- modal jako osobny stan,
- spinner albo progress przy taskach >100–200 ms,
- debounce dla częstych eventów,
- bezpieczne zamykanie goroutines.

---

## Jak skalować okno, aby wyświetlanie było estetyczne

### Zasada 1 — rozmiar przede wszystkim z `WindowSizeMsg`

Wymiary terminala powinny być aktualizowane przede wszystkim na podstawie:

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
```

### Zasada 2 — układ dziel na trzy warstwy

Praktyczny układ:

1. **Header / tabs**
2. **Body / workspace**
3. **Footer / shortcuts**

Jeżeli aplikacja używa legendy, statusu lub stałych oznaczeń kolorów, traktuj je jak część warstwy dolnej:

1. **Header / tabs**
2. **Body / workspace**
3. **Legend / status**
4. **Footer / shortcuts**

### Zasada 3 — body liczone z odejmowaniem header/footer

Przykład:

```go
innerWidth := max(20, m.width-4)
headerHeight := lipgloss.Height(header)
footerHeight := lipgloss.Height(footer)
availableHeight := m.height - headerHeight - footerHeight - 4
if availableHeight < 5 {
    availableHeight = 5
}
```

### Zasada 4 — styl definiuje wygląd, nie rozmiar bazowy

Styl typu `contentBoxStyle` powinien ustawiać:

- ramkę,
- kolory,
- padding,
- typ obramowania,

ale nie powinien narzucać sztywnego rozmiaru, jeśli aplikacja ma być responsywna.

### Zasada 5 — textarea i viewport muszą znać realną szerokość wnętrza

Uwzględnij:

- border left/right,
- padding left/right,
- marginesy rodzica.

Nie ustawiaj szerokości komponentu tylko na `m.width`.

### Zasada 6 — długie treści trzymaj w viewportach

Dla:

- logów,
- markdown,
- preview,
- dokumentacji,
- dumpów JSON,
- wyników procesów.

### Zasada 7 — ustaw minimalne i maksymalne limity

Przykłady:

```go
bodyWidth := max(40, m.width-4)
popupWidth := min(70, max(40, m.width/2))
textareaHeight := max(6, availableBodyHeight/3)
```

### Zasada 8 — testuj trzy rozmiary terminala

Każdy ekran powinien być sprawdzony co najmniej dla:

- małego terminala,
- średniego,
- szerokiego.

### Zasada 9 — status bar zawsze pełnej szerokości

Footer powinien zajmować całą szerokość obszaru roboczego.

Jeśli aplikacja wyświetla legendę, powinna ona również mieć stałe miejsce i nie powinna przewijać się razem z treścią. Przewijać powinno się tylko body/workspace.

### Zasada 10 — nie mnoż ramek bez potrzeby

Zbyt wiele ramek i zagnieżdżonych boxów zmniejsza realną przestrzeń na dane.

---

## Obsługa przycisków

### Model działania przycisku

Przycisk w TUI to zwykle:

- focusowalny element stanu,
- render inny, gdy ma fokus,
- aktywacja przez `Enter`,
- opcjonalnie aktywacja przez skrót globalny.

### Zasady implementacyjne

1. Każdy ekran z przyciskiem powinien mieć indeks fokusu.
2. Strzałki i `Tab` przełączają focus.
3. `Enter` uruchamia akcję tylko wtedy, gdy fokus jest na przycisku.
4. Styl aktywnego przycisku musi się różnić od nieaktywnego.
5. Dla akcji niszczących używaj potwierdzenia w modalu.

### Minimalny wzorzec

```go
if m.focusIndex == startButtonIndex && key == "enter" {
    m.loading = true
    return m, runJobCmd(input)
}
```

### Dobre praktyki UX dla przycisków

- główna akcja ma wyróżniony kolor,
- wtórna akcja ma kolor neutralny,
- `Esc` odpowiada anulowaniu,
- gdy trwa zadanie, przycisk powinien być zablokowany albo zamieniony w stan `busy`.

### Wzorzec przycisku start w formularzu konfiguracji

Jeżeli ekran konfiguracji ma działać jak w `DataDownloader`, `Analyzer` albo `HyperoptAssistant`, przycisk startu powinien być traktowany jako osobny fokusowalny element formularza, a nie tylko skrót w stopce.

Minimalny wzorzec implementacyjny:

1. Dodaj osobny typ pola submit, np. `configFieldSubmit` albo stały indeks typu `runtimeFieldSubmit`.
2. Dodaj to pole na końcu listy pól formularza, tak aby `Tab` i `Shift+Tab` naturalnie mogły na nie wejść.
3. Jeżeli model trzyma `inputs` równolegle do `fields`, dodaj także pusty wpis do `inputs`, żeby nie rozjechały się indeksy.
4. W `renderBody()` obsłuż to pole specjalnie:
   - nie renderuj zwykłego label + input,
   - renderuj obramowany przycisk z wyraźnym stanem focus,
   - zachowaj lekkie wcięcie, żeby przycisk był wyrównany z resztą formularza.
5. W `Update()` zostaw `Enter` na ostatnim fokusie jako akcję startu; nie twórz dla tego osobnej ścieżki, jeśli formularz już zapisuje na ostatnim polu.
6. W `footerHelpText()` pokaż jawny komunikat typu `Enter start ... | Ctrl+S start`.

Przykładowy helper renderujący:

```go
func renderStartButton(label string, focused bool) string {
    var btnStyle lipgloss.Style
    if focused {
        btnStyle = lipgloss.NewStyle().
            Bold(true).
            Padding(0, 3).
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.Color("40")).
            Foreground(lipgloss.Color("40"))
    } else {
        btnStyle = lipgloss.NewStyle().
            Padding(0, 3).
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.Color("238")).
            Foreground(lipgloss.Color("238"))
    }
    button := btnStyle.Render("  ▶   " + strings.ToUpper(strings.TrimSpace(label)) + "  ")
    const btnLeftOffset = 2
    lines := strings.Split(button, "\n")
    for index, line := range lines {
        lines[index] = strings.Repeat(" ", btnLeftOffset) + line
    }
    return strings.Join(lines, "\n")
}
```

Zasada UX:

- fokus na przycisku musi być natychmiast widoczny,
- przycisk powinien być częścią normalnej sekwencji `Tab`,
- `Ctrl+S` może pozostać szybkim skrótem startu, ale nie zastępuje widocznego przycisku,
- stopka musi zmieniać opis klawiszy po wejściu na pole submit.

---

## Wzorce repozytoryjne z 4 aplikacji

Poniższe wzorce są już realnie używane w `Analyzer`, `HyperoptAssistant`, `DataDownloader` i `FreqAIAssistant`. Traktuj je jako wzorce referencyjne dla nowych narzędzi z tej rodziny.

### 1. Jeden shell TUI na widok, nie doklejanie bloków

Wszystkie pełnoekranowe widoki powinny renderować się przez jeden wspólny shell:

- `header`
- `body`
- `footer`
- opcjonalnie modal wyświetlany wewnątrz tego samego shellu

Wzorzec:

- `renderShellHeader(...)`
- `renderAppShell(...)`
- `renderShellFooter(...)`
- `tea.View{AltScreen: true}`

Nie buduj widoku przez dopisywanie kolejnych "sekcji ekranu" do stdout. Każdy ekran ma być odświeżany jako jedna spójna rama.

### 2. Scrolluje się tylko body

W tych czterech aplikacjach przewijalny jest wyłącznie obszar roboczy:

- listy opcji,
- logi procesu,
- final review,
- długie formularze help / review.

Header, legenda, status i footer pozostają przypięte. Implementacyjnie oznacza to:

- `viewport.Model` tylko dla body,
- wysokość body liczona z `innerHeight - headerHeight - footerHeight`,
- brak przewijania całego shellu.

### 3. Formularz konfiguracji ma jawne typy pól

Konfiguracja startowa nie jest zbiorem luźnych promptów. Model formularza powinien mieć:

- listę pól,
- `focusIndex`,
- typ pola,
- walidację,
- hint,
- opcjonalny modal picker,
- opcjonalne pole submit.

W praktyce pole ma jeden z typów:

- zwykły tekst,
- picker pojedynczego wyboru,
- picker wielokrotnego wyboru,
- read-only pill,
- submit button.

To pozwala utrzymać stabilny layout bez rozgałęziania logiki po całym `Update()`.

### 4. Bounded picker zamiast wolnego tekstu

Gdy zbiór opcji jest ograniczony, używaj pickera zamiast ręcznego wpisywania tekstu.

Ten wzorzec jest już używany dla:

- strategii,
- modeli,
- presetów,
- configów freqtrade,
- trybów wejścia,
- przestrzeni hyperopt,
- loss functions,
- action planów,
- cleanup candidate selection,
- timeframes / candle types / bool options.

Standard pickera:

- sortowanie deterministyczne,
- `/` lub `Ctrl+F` otwiera filtr,
- `?` / `F1` otwiera help,
- `A-Z` robi first-letter jump,
- `Ctrl+A` / `Ctrl+X` działa w trybie multi-select,
- `allowEmpty` jest jawne i nie wynika pośrednio z pustej listy.

### 5. Modal picker blokuje tło

Gdy picker lub dialog jest otwarty:

- formularz tła nie dostaje inputu,
- `Esc` zamyka modal,
- `Enter` zatwierdza,
- filtr działa tylko w modalu,
- rozmiar modalu jest liczony względem aktywnego shellu, nie całego ekranu "na sztywno".

Wzorzec implementacyjny:

- `if m.modal != nil { return m.updateModal(msg) }`
- osobny `pickerModalModel`
- `View(innerWidth, bodyHeight)` dla modala

### 6. Fallback z convenience UI do edycji ręcznej

W tych narzędziach picker lub discovery jest wygodą operatora, a nie jedyną ścieżką działania.

Stosowana reguła:

- jeśli discovery strategii/modeli/configów się nie powiedzie,
- UI degraduje się do ręcznej edycji pola,
- sesja nie kończy się błędem tylko dlatego, że convenience layer była niedostępna.

Przykład:

- `Analyzer`: awaria discovery pickera strategii nie może zablokować ręcznej edycji `strategy_filter`.

### 7. Readiness scan filtruje wejście przed wykonaniem

Zanim użytkownik wybierze strategie do uruchomienia, aplikacja powinna wykonać scan/readiness pass i pokazać tylko elementy bezpieczne dla danego workflow.

Ten wzorzec występuje jako:

- `Analyzer`: strategia dostępna dla backtestu,
- `HyperoptAssistant`: strategia gotowa dla wybranych spaces,
- `DataDownloader`: strategie tylko do rozszerzenia planu timeframe,
- `FreqAIAssistant`: tylko `READY` i `READY_WITH_WARNINGS`.

Ważna reguła implementacyjna:

- wynik readiness jest osobną strukturą danych,
- picker pokazuje tylko dopuszczalne pozycje,
- niedopuszczalne pozycje są raportowane, ale nie rozwalają sesji.

### 8. Runner local-first, docker-second

Wspólny wzorzec runnera:

1. Spróbuj rozwiązać lokalny executable.
2. Jeśli to się nie uda i komenda to bare `freqtrade`, przejdź do Docker Compose fallback.
3. W Dockerze:
   - znajdź plik compose,
   - ustal service,
   - jeśli `docker_service` istnieje, traktuj go jako jawny wybór,
   - jeśli nie, próbuj heurystyk dopiero potem,
   - upewnij się, że service działa,
   - ustaw `TranslatePath`.

`commandRunner` powinien przenosić razem:

- `Mode`,
- `Executable`,
- `PrefixArgs`,
- `WorkingDir`,
- `DockerService`,
- `HostUserDataRoot`,
- `ContainerUserDataRoot`,
- `TranslatePath`.

### 9. Path translation musi być jawne i ograniczone do `user_data`

W ścieżce Docker fallback hostowe ścieżki nie mogą być tłumaczone "na oko". Wzorzec używany w repo:

- `TranslatePath(path string) (string, error)`
- `filepath.Rel(hostUserDataRoot, path)`
- błąd, jeśli ścieżka wychodzi poza mountowany root

To jest szczególnie ważne dla:

- przygotowanych configów,
- exportów,
- skryptów launch,
- logów,
- runtime artifact roots.

### 10. Prepared runtime config zamiast modyfikacji configu bazowego

`DataDownloader` i `FreqAIAssistant` stosują wzorzec "prepared config snapshot":

- wczytaj config bazowy,
- nadpisz tylko runtime overrides,
- zapisz wynik do osobnego pliku,
- uruchamiaj `freqtrade` na tym pliku,
- nie nadpisuj configu bazowego jako efekt uboczny wykonania.

Jeżeli prepared config jest częścią produktu, raportuj również:

- gdzie został zapisany,
- jakie override zostały nałożone,
- hash bazowego i wynikowego configu, jeśli workflow tego wymaga.

### 11. Runtime state przez tracker + snapshot

Dashboard nie powinien czytać bezpośrednio rozproszonych pól z wielu goroutines. Wspólny wzorzec:

- tracker przechowuje stan mutowalny pod mutexem,
- dashboard woła `Snapshot()`,
- renderer pracuje na niemutowalnej kopii,
- eventy są append-only,
- wynik kroku jest zapisywany raz przez `Finish(...)`.

To upraszcza:

- odświeżanie,
- stop request,
- aktywny proces,
- log tail,
- render końcowych wyników.

### 12. Stop request jest osobnym stanem, nie tylko sygnałem procesu

W tych programach zatrzymanie sesji nie jest utożsamione z `Kill`.

Używany wzorzec:

- `RequestStop(...)` zapisuje intencję operatora,
- UI pokazuje `stop requested`,
- feeder przestaje podawać nowe zadania,
- aktywny proces jest zatrzymywany kontrolowanie,
- final summary nadal się generuje.

To rozdziela:

- intencję stopu,
- stan dashboardu,
- faktyczne zakończenie procesu,
- wynik końcowy (`OK`, `ABORT`, `ERROR`, `SKIP`).

### 13. Output procesu trafia do trackera i raportu, nie bezpośrednio do stdout

Wspólny wzorzec:

- subprocess stdout/stderr są przechwytywane,
- ostatnie linie trafiają do dashboardu,
- pełny kontekst może trafić do pliku logu lub podsumowania,
- aktywny TUI nie jest zanieczyszczany surowym `fmt.Println`.

### 14. Atomic write jako domyślny zapis stanu trwałego

Wszystkie trwałe artefakty użytkowe powinny być zapisywane przez:

- `CreateTemp(...)`
- zapis do pliku tymczasowego
- `Close()`
- `Rename()` / replace

To dotyczy:

- reportów TXT,
- JSON history,
- cache,
- prepared configów,
- preset stores,
- audit outputs.

Na Windows trzeba uwzględnić wariant `remove + rename`, jeśli bezpośredni `Rename()` na istniejący target nie działa.

### 15. Corrupt persistence nie może blokować programu

`Analyzer`, `HyperoptAssistant` i `FreqAIAssistant` pokazują wzorzec odporności:

- jeśli cache/history jest uszkodzony, ostrzeż operatora,
- spróbuj odtworzyć bezpieczny pusty stan,
- opcjonalnie przenieś uszkodzony plik do `.corrupt.*`,
- nie kończ sesji, jeśli produkt może działać dalej.

### 16. Cache/history muszą być keyed by real execution identity

Wspólny wzorzec nie może opierać się tylko na nazwie strategii. Klucz powinien uwzględniać realne parametry wykonania:

- checksum pliku strategii,
- config checksum / prepared config hash,
- timerange,
- pair set lub pair timeranges,
- model / identifier / stage, jeśli workflow tego wymaga,
- ważne runtime args.

Wzorzec:

- exact-match reuse tylko dla pełnego podpisu wykonania,
- near-match differences jako osobny audit/report,
- reused result oznaczony jawnie jako cache reuse.

### 17. Report i live UI są osobnymi produktami

To repo już konsekwentnie rozdziela:

- live dashboard do monitoringu,
- TXT report do trwałego podsumowania,
- JSON cache/history do mechaniki produktu.

Zmiana układu TUI nie może automatycznie zmieniać formatu raportu końcowego. Renderery powinny być oddzielone od modeli domenowych.

### 18. Sortowanie końcowych wyników musi być deterministyczne

Nie wolno polegać na kolejności zakończenia workerów ani kolejności map.

Sortowanie powinno być jawne i stabilne dla:

- końcowego rankingu,
- list wyników w dashboardzie,
- wpisów w raporcie,
- par / rodzin identifierów / comparable runs / cleanup candidates.

### 19. Runtime artifacts mają własny katalog produktu pod `user_data`

Każda aplikacja, która tworzy runtime artifacts, powinna trzymać je pod własnym katalogiem produktu, zamiast rozrzucać pliki po root repo.

Wzorzec już obecny:

- `user_data/datadownloader`
- `user_data/freqaiassistant`

Jeżeli aplikacja zapisuje:

- prepared config,
- export archives,
- launch scripts,
- handoff guides,
- session reports,
- session history,

to powinny one być grupowane pod jednym dedykowanym rootem.

### 20. Cleanup i maintenance zaczynają się od audytu read-only

`FreqAIAssistant` wprowadza dobry wzorzec dla operacji potencjalnie destrukcyjnych:

1. najpierw audit i lista kandydatów,
2. potem jawny picker potwierdzenia,
3. potem cleanup tylko wewnątrz zadanego rootu,
4. opcjonalnie spójne sprzątnięcie wpisu history,
5. finalny wynik cleanupu w raporcie.

Nie łącz detekcji kandydata i automatycznego usuwania w jeden krok.

### 21. Audyty doradcze nie powinny blokować trybu audit-only

W tych aplikacjach nie każda niepewność jest błędem blokującym. Dobrą praktyką jest rozdzielenie:

- `blocking errors`,
- `risky conditions`,
- `advisory findings`.

Przykłady auditów doradczych:

- local data coverage,
- identifier freshness,
- artifact completeness,
- runtime path warnings,
- picker discovery limits.

Tylko warunki rzeczywiście uniemożliwiające wykonanie powinny zatrzymywać workflow.

### 22. Final review powinien być osobnym ekranem, nie tylko końcem logu

`FreqAIAssistant` pokazuje dodatkowy wzorzec:

- po wykonaniu sesji można otworzyć osobny ekran review,
- ranking, gate, audit i recommendations są renderowane w jednym przewijalnym widoku,
- ekran review używa tego samego shellu TUI co reszta aplikacji, ale ma własny content model.

Jeśli produkt ma złożone wnioski końcowe, nie upychaj ich wyłącznie do stopki albo ostatnich eventów dashboardu.

### 23. Launch / handoff artifacts dla procesów operacyjnych

Jeżeli aplikacja przygotowuje uruchomienie czegoś poza swoim procesem:

- wygeneruj skrypt launch,
- wygeneruj ścieżkę logu,
- wygeneruj instrukcję attach/follow,
- pokaż to w dashboardzie i zapisz do pliku handoff.

To jest lepszy wzorzec niż "wypisz komendę do skopiowania" i zostaw operatora bez dalszego kontekstu.

### 24. Skanowanie repo ma mieć listę wykluczeń narzędziowych

Wzorzec obecny we wszystkich czterech aplikacjach:

- nie skanuj `.git`,
- nie skanuj `.venv` / `venv`,
- nie skanuj `__pycache__`,
- nie skanuj katalogów innych narzędzi z tej samej rodziny, gdy scan ma dotyczyć strategii użytkownika.

To powinno być centralnie udokumentowane i trzymane deterministycznie, żeby uniknąć szumu i fałszywych trafień.

---

## Obsługa pól wejściowych

### Podstawowe zasady

1. Każde pole ma własny stan.
2. Focus jest jawnie zarządzany.
3. Walidacja odbywa się:
   - live,
   - przy utracie fokusu,
   - przy submit.
4. Błąd walidacji musi mieć własny kolor i stałe miejsce.
5. Placeholder nie może zastępować labela.

### Dobre praktyki

- osobny label nad polem,
- placeholder tylko jako podpowiedź,
- walidacja wyświetlana zaraz pod polem,
- `Enter` przechodzi do następnego pola lub wykonuje akcję,
- dla wielu pól rozważ `Huh` zamiast ręcznej implementacji.
- domyślnie preferuj gotowe komponenty z `bubbles` lub `huh` zamiast własnych ad-hoc inputów.

### Typy wejścia

#### Tekst
- zwykły input, najlepiej oparty o `bubbles/textinput`,
- limit długości,
- przycinanie niepoprawnych znaków, jeśli trzeba.

#### Liczby
- walidacja `Atoi`, `ParseFloat` lub regex,
- komunikat o błędzie pod polem,
- ewentualnie automatyczne czyszczenie niedozwolonych znaków.

#### Daty
- walidacja `time.Parse("2006-01-02", value)`,
- przycisk główny zablokowany albo ostrzeżony, gdy format jest błędny.

#### Wielolinijkowe notatki
- `textarea`, najlepiej oparta o `bubbles/textarea`,
- ograniczony viewport,
- licznik znaków i linii mile widziany,
- preview powinno być skracane.

---

## Obsługa danych wejściowych i akcji

### Architektura zdarzeń

Zalecany przepływ:

1. User input → `tea.KeyPressMsg`
2. `Update()`:
   - aktualizacja fokusu,
   - aktualizacja konkretnego komponentu,
   - uruchomienie `Cmd` jeśli potrzeba.
3. `Cmd` zwraca wiadomość końcową.
4. `Update()` odbiera wynik i aktualizuje stan.
5. `View()` renderuje nowy ekran (w Bubble Tea v2 zwraca `tea.View`, nie `string`).

**Uwaga Bubble Tea v2 — typy zdarzeń klawiatury:**
- `tea.KeyPressMsg` — preferowany typ w v2; zawiera `Code`, `Mod`, `Text`.
- `tea.KeyMsg` — alias wstecznej kompatybilności; działa tak samo jak `KeyPressMsg`.
- W nowym kodzie używaj `tea.KeyPressMsg`; `tea.KeyMsg` jest dostępne dla migracji z v1.
- Sprawdzaj wartość klawisza przez `msg.String()` (np. `"enter"`, `"ctrl+c"`) lub przez pola `msg.Code` i `msg.Mod`.

### Dane powinny być rozdzielone od widoków

Nie trzymaj formatowanego tekstu jako źródła prawdy. Trzymaj:

- struct z danymi,
- status walidacji,
- wynik operacji,
- dopiero potem renderuj.

### Przykład wzorca

```go
type SchedulerState struct {
    FromDate string
    ToDate   string
    Valid    bool
    Busy     bool
}
```

---

## Skróty na dole okna

Dolny pasek skrótów to jeden z najważniejszych elementów dobrej aplikacji TUI.

### Co powinien zawierać

- najważniejsze skróty globalne,
- skróty kontekstowe dla aktywnej zakładki,
- informację o aktywnym trybie,
- ewentualnie stan filtrowania albo zaznaczenia.

### Przykładowy układ

```text
←/→ tabs • Tab next focus • Shift+Tab prev focus • Enter activate • Esc close • q quit
```

### Zasady

1. Pasek musi być zawsze widoczny.
2. Powinien zmieniać się zależnie od aktywnego panelu.
3. Nie może być zbyt długi — priorytet dla najważniejszych akcji.
4. Skróty muszą odpowiadać rzeczywistej implementacji.
5. Dla modala footer może być zastąpiony przez skróty modalowe.

### Praktyczny wzorzec

- globalne skróty: `q`, `Esc`, `Tab`, `←/→`
- panelowe skróty: `Enter`, `/`, `s`, `d`, `r`
- modalowe skróty: `Enter`, `Esc`, `←/→` lub `Tab`

---

## Zalecana struktura projektu

```text
cmd/app/main.go
internal/app/model.go
internal/app/update.go
internal/app/view.go
internal/app/styles.go
internal/app/tabs/dashboard.go
internal/app/tabs/logs.go
internal/app/tabs/form.go
internal/app/tabs/notes.go
internal/app/tabs/scheduler.go
internal/app/modals/modal.go
internal/app/components/footer.go
internal/app/components/header.go
internal/app/components/buttons.go
internal/app/components/inputs.go
internal/app/domain/... 
internal/app/services/... 
```

### Zasady podziału

- `main.go` — bootstrap,
- `model.go` — stan główny,
- `update.go` — routing eventów,
- `view.go` — widok główny,
- `styles.go` — paleta i style,
- `tabs/*` — logika i widoki per zakładka,
- `modals/*` — popupy i dialogi,
- `services/*` — sieć, pliki, procesy, API.

---

## Kiedy używać konkretnych bibliotek

### Bubble Tea + Bubbles + Lip Gloss
Używaj jako domyślnego stacku dla pełnych TUI.

W praktyce:

- `Bubble Tea` jako runtime i model zdarzeń,
- `Bubbles` dla gotowych inputów, list, viewportów, spinnerów i komponentów fokusowalnych,
- `Lip Gloss` do ramki, layoutu, status bara, footerów i legendy.

### Huh
Używaj, gdy formularz ma:

- wiele pól,
- wiele kroków,
- selekty,
- walidację,
- tryb accessible.

Jeżeli formularz da się naturalnie zbudować z `Huh`, nie twórz równoległego własnego frameworka promptów.

### Glamour
Używaj dla:

- help screen,
- README view,
- docs,
- changelog,
- release notes.

### Wish
Używaj, gdy aplikacja ma działać po SSH.

### go-pretty/table
Używaj dla:

- tabel statusu,
- raportów,
- porównań,
- wyników batchowych.

### go-pretty/progress
Używaj dla:

- długich jobów,
- importów,
- synchronizacji,
- skanowania,
- build pipeline’ów.

### Cobra + Viper
Używaj, gdy aplikacja ma mieć także klasyczne CLI:

- `app run`
- `app config`
- `app doctor`
- `app export`
- `app server`

TUI może być wtedy jedną z podkomend.

### PTerm
Używaj, gdy nie potrzebujesz pełnego TUI, tylko nowoczesnego klasycznego CLI.

### tcell
Używaj tylko wtedy, gdy potrzebujesz niskopoziomowej kontroli nad ekranem lub własnego renderera.

---

## Reguły dla Codex i GitHub Copilot

Poniższa sekcja jest celowo napisana jak specyfikacja implementacyjna.

### Global rules

- Prefer **Bubble Tea v2**, **Bubbles v2**, and **Lip Gloss v2** together as the default stack for new full-screen TUIs.
- Do not mix legacy `github.com/charmbracelet/...` imports with `charm.land/.../v2` imports in the same project.
- Keep a single root model for app state.
- Handle `tea.WindowSizeMsg` and recompute all dimensions on resize.
- Keep `View()` pure: no network, no sleeps, no heavy loops, no side effects.
- **Uwaga Bubble Tea v2:** `View()` returns `tea.View`, not `string`. Use `model.View().Content` to access the rendered string. Set `tea.View.AltScreen = true` to enable alternate screen for full-screen apps.
- Use `tea.Cmd` for long-running work.
- Render long content inside `viewport` or truncate previews.
- Never print debug logs to stdout while the TUI is running.
- Keep shortcuts visible in a footer/status bar.
- Use modal state to block background input while dialogs are open.

### Freeze prevention rules

- Never call blocking I/O directly in `Update()`.
- Never render two full-screen views by concatenating strings.
- Never let textarea preview grow unbounded.
- Use `loading`, `busy`, and `error` flags explicitly.
- Add progress or spinner for operations longer than a fraction of a second.
- Route background results back into the app via typed messages.

### Layout rules

- Compute `headerHeight`, `footerHeight`, and `availableBodyHeight` every render.
- Derive child widths from the real inner width of the parent container.
- Use min/max guards for widths and heights.
- Use a viewport for logs, markdown, JSON, and long descriptions.
- Keep footer always visible.
- Keep legend/status pinned when present; only the workspace/body should scroll.
- Prefer one clear outer frame or container for the app shell.

### Input rules

- Each input field must have a label, current value, validation status, and focus style.
- `Tab` and `Shift+Tab` move focus.
- `Enter` activates the focused control.
- `Esc` closes the modal or cancels the current action.
- Date inputs should validate against an explicit, documented format such as `2006-01-02`, unless the product has a stronger domain-specific convention.
- Numeric inputs should validate with parsing, not only with placeholders.
- Prefer `bubbles` and `huh` input components over custom one-off input handling unless the product has a strong reason to diverge.

### Modal rules

- Modal is a dedicated render mode.
- Background controls do not receive input while modal is open.
- `Esc` closes modal.
- `Enter` confirms.
- Support button focus for multi-action dialogs.

### Footer rules

- Footer must always show the currently valid shortcuts.
- Footer content should change with the active tab or modal.
- Keep it compact and truthful.
- Footer should stay pinned to the bottom of the visible app shell.
- If a legend is present, it should also stay pinned in its own stable area instead of moving with scrollable content.

---

## Przykładowe standardy UX

### Standard globalny

- `q` — quit/back zależnie od kontekstu
- `Ctrl+C` — interrupt/quit zależnie od klasy aplikacji
- `Tab` — next focus
- `Shift+Tab` — previous focus
- `←` / `→` — switch tabs
- `Esc` — close/cancel/back
- `Enter` — primary action
- `?` — help

### Standard formularza

- `Enter` przechodzi do kolejnego pola albo submituje,
- błędy walidacji są pod polami,
- submit nie powinien udawać sukcesu, gdy dane są błędne.

### Standard schedulera

- daty domyślne ustawione na sensowne wartości,
- walidacja dat live,
- przycisk `Start` pokazuje stan busy albo modal potwierdzenia,
- błąd zakresu dat widoczny natychmiast.

### Standard logów

- viewport,
- możliwość przewijania,
- przycisk/skrót clear,
- highlight dla poziomów logów.

---

## Lista rekomendowanych zależności dla nowego projektu

### Minimalny zestaw TUI

```bash
go get charm.land/bubbletea/v2
go get charm.land/bubbles/v2
go get charm.land/lipgloss/v2
```

### Formularze

```bash
go get charm.land/huh/v2
```

### Dokumentacja (Markdown render) — wybór linii

```bash
# Stabilna v0.x — szeroka kompatybilność, nie wymaga całego stosu charm.land
go get github.com/charmbracelet/glamour

# v2 — nowe API, wymaga całego stosu charm.land v2
go get charm.land/glamour/v2
```

### SSH

```bash
go get github.com/charmbracelet/wish
```

### Klasyczne CLI i konfiguracja

```bash
go get github.com/spf13/cobra
go get github.com/spf13/viper
```

### Tabele i postęp

```bash
go get github.com/jedib0t/go-pretty/v6/table
go get github.com/jedib0t/go-pretty/v6/progress
go get github.com/jedib0t/go-pretty/v6/text
```

### Logowanie (charm.land stack)

```bash
go get charm.land/log/v2
```

### Detekcja profilu kolorów (v2 stack)

```bash
go get github.com/charmbracelet/colorprofile
```

### Dodatki

```bash
go get github.com/pterm/pterm
go get github.com/muesli/termenv
go get github.com/muesli/reflow
go get github.com/gdamore/tcell/v2
go get github.com/nao1215/prompt
```

---

## Wzorce współbieżności w aplikacjach CLI

Ta sekcja dotyczy wzorców Go dla bezpiecznej współbieżności w aplikacjach CLI i TUI.

### Wzorzec feeder — unikanie wycieku goroutine

Goroutina "feeder" wysyłająca zadania do niebuforowanego kanału **musi** używać `select` z kontekstem zatrzymania. Bez tego, gdy workerzy wyjdą wcześniej, feeder zablokuje się na `jobs <- item` na zawsze.

**Niepoprawne (wyciek goroutiny):**
```go
go func() {
    defer close(jobs)
    for _, item := range pending {
        jobs <- item  // blokuje na zawsze gdy workery zakończą
    }
}()
```

**Poprawne (z context.Done):**
```go
go func() {
    defer close(jobs)
    var stopDone <-chan struct{}
    if stop != nil {
        stopDone = stop.Context().Done()
    }
    for _, item := range pending {
        if stop != nil && stop.Requested() {
            return
        }
        select {
        case jobs <- item:
        case <-stopDone:
            return
        }
    }
}()
```

> **Uwaga:** nil channel w `select` nigdy nie odpala — jeśli `stopDone == nil`, gałąź `case <-stopDone` jest ignorowana, co jest zachowaniem poprawnym.

---

### Nil kanał w select — opcjonalne ścieżki bez `if`

W Go, **odczyt z nil kanału (`<-nilChan`) blokuje na zawsze** — gałąź `select` nigdy nie odpala. To przydatny wzorzec do tworzenia opcjonalnych ścieżek przerwania bez warunków `if` rozsianych po całym kodzie.

```go
// Opcjonalny stopDone — gdy stop == nil, gałąź case <-stopDone nigdy nie odpala.
var stopDone <-chan struct{}
if stop != nil {
    stopDone = stop.Context().Done()
}

for _, item := range pending {
    select {
    case jobs <- item:
    case <-stopDone:  // nil → ta gałąź zawsze blokuje → nigdy nie odpala
        return
    }
}
```

Kiedy stosować:

- Opcjonalny stop-controller — aplikacja może działać bez mechanizmu zatrzymania
- Opcjonalny timeout — timeout tylko gdy konfiguracja go przewiduje
- Opcjonalna obsługa anulowania — nil kontekst oznacza "bez deadline"

```go
// Opcjonalny timeout — ctx nil oznacza "bez ograniczenia czasu"
var deadline <-chan struct{}
if ctx != nil {
    deadline = ctx.Done()
}

select {
case result := <-resultCh:
    process(result)
case <-deadline:
    return ErrTimeout
}
```

> **Uwaga:** Wzorzec `var ch <-chan T` (tylko odczyt) zawsze inicjalizuje się jako `nil`. Przypisanie `chan T` do `<-chan T` działa automatycznie.

---

### sync.Once do jednorazowej inicjalizacji zasobu

Gdy zasób (np. program TUI) może być tworzony z wielu goroutyn, **nie używaj ręcznego podwójnego sprawdzenia** (`check → unlock → create → lock → check again`). Między pierwszym unlock a drugim lock dwie goroutyny mogą jednocześnie przejść przez sprawdzenie nil i obie uruchomić zasób.

**Niepoprawne (double-checked locking — ryzyko dwóch programów TUI):**
```go
mu.Lock()
if ui == nil {
    mu.Unlock()
    ui = newUI()    // obie goroutyny mogą tu dotrzeć jednocześnie
    mu.Lock()
    if ui == nil {
        ...
    }
}
mu.Unlock()
```

**Poprawne (sync.Once):**
```go
type Tracker struct {
    uiOnce sync.Once
    mu     sync.Mutex
    ui     *LiveUI
}

func (t *Tracker) ensureUI() {
    t.uiOnce.Do(func() {
        ui := newLiveUI()
        t.mu.Lock()
        t.ui = ui
        t.mu.Unlock()
    })
}
```

> `sync.Once.Do` blokuje wszystkich innych callerów do czasu zakończenia pierwszego wykonania. Gwarantuje dokładnie jedno wywołanie nawet przy dowolnej liczbie goroutyn.

---

### Odczyt pola wskaźnikowego bez blokady to wyścig danych

Pole wskaźnikowe chronione przez mutex **musi być odczytywane pod blokadą**, nawet jeśli w praktyce wywołanie jest zawsze sekwencyjne.

```go
// BŁĄD — wyścig danych wykrywalny przez go test -race
func (p *Tracker) Close() {
    if p.ui != nil {    // odczyt bez p.mu
        p.ui.Close()
    }
}

// POPRAWNIE — snapshot pod blokadą
func (p *Tracker) Close() {
    p.mu.Lock()
    ui := p.ui
    p.mu.Unlock()
    if ui != nil {
        ui.Close()
    }
}
```

---

### Bubble Tea — bezpieczeństwo wątkowe

- `program.Send(msg)` jest **goroutine-safe** — można wywoływać z dowolnej goroutyny w dowolnym czasie.
- `Update()` i `View()` są zawsze wywoływane **sekwencyjnie** przez Bubble Tea — nie są goroutine-safe między sobą, ale nie muszą być, bo BT gwarantuje kolejność.
- Nie wolno trzymać blokady mutexa podczas wywołania `ui.Refresh()` ani `program.Send()` — mogłoby to spowodować deadlock jeśli Bubble Tea spróbuje wywołać z powrotem funkcję wymagającą tej samej blokady.

---

### Wzorzec graceful stop w puli workerów

Zatrzymanie workerów powinno być **graceful**: bieżące zadanie zawsze dobiega końca. Sprawdzanie `stop.Requested()` następuje **po** zakończeniu zadania, nie przed jego startem.

```go
for strategy := range jobs {
    result := doWork(strategy)
    output <- result
    // Graceful stop: bieżące zadanie zawsze dobiega końca.
    // Feeder nie wyśle nowych zadań gdy stop jest aktywny.
    if stop != nil && stop.Requested() {
        return
    }
}
```

---

### Buforowanie kanału wyjścia (output channel)

Niebuforowany kanał wyjścia powoduje, że worker **blokuje się** na `output <- result` dopóki konsument nie odczyta wartości. Jeśli konsument wykonuje wolną operację (np. zapis pliku), wszystkie workery stoją bezczynnie.

```go
// ŹLE — niebuforowany: każdy worker czeka na konsumenta
output := make(chan StrategyResult)

// DOBRZE — bufor = workerCount: każdy worker może "wyprzedzić" konsumenta
output := make(chan StrategyResult, workerCount)
```

Zasada doboru rozmiaru bufora:

| Scenariusz | Rozmiar bufora |
|---|---|
| Konsument szybki (goroutyna, kanał) | `0` (niebuforowany) |
| Konsument wolny (I/O, blokada) | `workerCount` |
| Znane maksimum kolejki | dokładna liczba |

Bufor `workerCount` gwarantuje, że **żaden** worker nie zablokuje się tylko dlatego, że konsument przetwarza poprzedni wynik.

---

### Pułapka: mutex trzymany podczas I/O

**Nie trzymaj blokady podczas długich operacji I/O** (zapis pliku, zapytanie HTTP, itp.). Blokuje wszystkie goroutyny czekające na ten mutex.

```go
// RYZYKOWNE — trzyma mu podczas zapisu pliku
func (h *Store) Save() error {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return writeJSONAtomic(h.path, h.file)  // I/O pod blokadą
}

// LEPIEJ (gdy mapa jest duża) — snapshot pod blokadą, I/O poza
func (h *Store) Save() error {
    h.mu.RLock()
    snapshot := cloneEntries(h.file.Entries)  // szybka kopia referencji
    path := h.path
    h.mu.RUnlock()
    return writeJSONAtomic(path, snapshot)    // I/O poza blokadą
}
```

---

### Wzorzec: dwa mutexy — dane i zapis pliku rozdzielone

Gdy wiele goroutyn może wywoływać `Save()` jednocześnie, jeden mutex dla danych nie wystarczy — równoległe zapisy do tego samego pliku tworzą wyścig na poziomie systemu plików (zwłaszcza na Windows, gdzie `rename` wymaga wyłącznego dostępu).

Rozwiązanie: osobny mutex serializujący tylko operację zapisu, niezależny od mutexa chroniącego dane w pamięci.

```go
type historyStore struct {
    mu     sync.RWMutex // chroni h.file (dane w pamięci)
    saveMu sync.Mutex   // serializuje zapis pliku (syscall-level)
    path   string
    file   historyFile
}

func (h *historyStore) Save() error {
    if h == nil {
        return nil
    }
    // 1. Krótka blokada — tylko marshalowanie danych do []byte.
    //    Store() może działać równolegle po zwolnieniu RLock.
    h.mu.RLock()
    data, err := json.MarshalIndent(h.file, "", "  ")
    h.mu.RUnlock()
    if err != nil {
        return fmt.Errorf("cannot serialize history: %w", err)
    }
    // 2. saveMu serializuje zapis — żadne dwie goroutyny nie wykonują
    //    rename na tym samym pliku jednocześnie.
    h.saveMu.Lock()
    defer h.saveMu.Unlock()
    return writeDataAtomic(h.path, data)
}
```

Kluczowe cechy tego wzorca:
- `mu.RLock` trzymany tylko przez czas serializacji JSON (mikrosekundy)
- `Store()` może działać równolegle po zwolnieniu `mu.RLock`
- `saveMu` zapobiega równoległemu `rename` na tym samym pliku
- Oba mutexy są niezależne — brak ryzyka zakleszczenia

---

## Krótkie podsumowanie architektoniczne

Nowoczesna aplikacja konsolowa w Go powinna:

- mieć centralny model stanu,
- używać nieblokującego `Update()` i `tea.Cmd` dla cięższych zadań,
- skalować layout dynamicznie z `WindowSizeMsg`,
- renderować długie treści przez `viewport` lub limity preview,
- mieć spójny system focusu,
- pokazywać skróty w stopce,
- rozróżniać loading/error/empty/success,
- logować debug do pliku,
- używać aktywnie rozwijanych bibliotek zamiast starych prompt frameworków.

---

## Źródła

- Bubble Tea repo: https://github.com/charmbracelet/bubbletea
- Bubble Tea releases / v2 info: https://github.com/charmbracelet/bubbletea/releases
- Bubble Tea v2 package docs: https://pkg.go.dev/charm.land/bubbletea/v2
- Bubble Tea v2 upgrade guide: https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md
- Charm v2 announcement: https://charm.land/blog/v2/
- Bubbles releases: https://github.com/charmbracelet/bubbles/releases
- Bubbles repo: https://github.com/charmbracelet/bubbles
- Bubbles v2 package docs: https://pkg.go.dev/charm.land/bubbles/v2
- Lip Gloss repo: https://github.com/charmbracelet/lipgloss
- Lip Gloss releases / v2 guidance: https://github.com/charmbracelet/lipgloss/releases
- Lip Gloss v2 package docs: https://pkg.go.dev/charm.land/lipgloss/v2
- Huh repo: https://github.com/charmbracelet/huh
- Huh releases: https://github.com/charmbracelet/huh/releases
- Huh package docs: https://pkg.go.dev/charm.land/huh/v2
- Glamour repo: https://github.com/charmbracelet/glamour
- Glamour releases: https://github.com/charmbracelet/glamour/releases
- Wish repo: https://github.com/charmbracelet/wish
- Wish releases: https://github.com/charmbracelet/wish/releases
- go-pretty repo: https://github.com/jedib0t/go-pretty
- Cobra releases: https://github.com/spf13/cobra/releases
- Cobra user guide: https://github.com/spf13/cobra/blob/main/site/content/user_guide.md
- Cobra CLI generator: https://github.com/spf13/cobra-cli
- Viper repo: https://github.com/spf13/viper
- PTerm repo: https://github.com/pterm/pterm
- termenv repo: https://github.com/muesli/termenv
- tcell repo: https://github.com/gdamore/tcell
- tcell tutorial: https://github.com/gdamore/tcell/blob/main/TUTORIAL.md
- reflow repo: https://github.com/muesli/reflow
- nao1215/prompt repo: https://github.com/nao1215/prompt
- briandowns/spinner repo: https://github.com/briandowns/spinner
- survey repo: https://github.com/AlecAivazis/survey
- c-bata/go-prompt repo: https://github.com/c-bata/go-prompt
