---
name: dead-code-hunter
description: >
  Systematically detects and safely removes dead code from a repository —
  one unit per iteration, verified by compilation or tests before every commit.
  Use when asked to: remove dead code, clean up unused functions/variables/imports,
  remove commented-out code blocks, reduce code debt, purge unreachable code.
  Do NOT use for: refactoring, renaming, restructuring, or any change that alters
  program behavior.
  Invoke explicitly with $dead-code-hunter or implicitly when prompt matches.
---

# SKILL: DEAD CODE HUNTER

You are a senior engineer performing disciplined dead code removal.
Goal: Remove exactly one dead code unit per iteration. Verify. Commit. Never break the build.

---

## Startup Sequence

1. Check for `.dead-code-hunter.yml` in repo root — load if present.
2. Check for inline `SCOPE:` or `FOCUS:` in user prompt — apply as highest priority.
3. Apply defaults for remaining parameters.
4. Start immediately — never wait for input.

> **Priority order:** Inline context → `.dead-code-hunter.yml` → defaults

---

## Parameters

| Parameter                   | Default                          | Description                                              |
|----------------------------|----------------------------------|----------------------------------------------------------|
| `ITERATIONS`               | `10`                             | Number of iterations to run                              |
| `SCOPE`                    | `""`                             | Directory/module to scan; empty = entire repo            |
| `CERTAINTY`                | `safe`                           | `safe` = certain cases only; `all` = include caution     |
| `VALIDATION`               | `compile`                        | `compile` \| `test`                                      |
| `MAX_REMOVALS_PER_ITER`    | `1`                              | Dead code units removed per iteration                    |
| `COMMENTED_CODE_MIN_LINES` | `3`                              | Min lines in commented block to qualify for removal      |
| `STOP_ON_CONSECUTIVE_FAILS`| `2`                              | Halt after N consecutive build/test failures             |
| `OUTPUT_FORMAT`            | `human`                          | `human` \| `compact` \| `json`                           |
| `LOG_FILE`                 | `.dead-code-hunter.log`          | Append-mode log; `""` to disable                         |
| `PUSH_AFTER_COMMIT`        | `true`                           | git push mandatory after every commit                    |
| `PUSH_FAILURE_ACTION`      | `halt`                           | `halt` \| `retry`                                        |
| `FORBIDDEN`                | `["deps","vendor","generated"]`  | Paths/patterns never to touch                            |

---

## Dead Code Classification

### SAFE — remove without confirmation

| Category | Examples |
|---|---|
| Unused function / method | Zero call sites across entire repo |
| Unused local variable | Declared, never read |
| Unused import / include | Not referenced in file |
| Unused constant / macro | Defined, never used |
| Unreachable code | Code after `return` / `break` / `goto` |
| Commented-out code block | `//` or `/* */` blocks ≥ `COMMENTED_CODE_MIN_LINES` lines |

### CAUTION — skip if `CERTAINTY=safe`, include if `CERTAINTY=all`

| Category | Reason |
|---|---|
| Unused public API | May be consumed by external callers |
| Feature-flag guarded code | May be activated by config |
| Functions used only in tests | Test utility, not dead |
| `__attribute__((unused))` / `[[maybe_unused]]` | Intentionally suppressed |

### NEVER touch

- Files / paths in `FORBIDDEN`
- Public API in `include/` unless SCOPE explicitly targets it
- Any code annotated with `// keep`, `// intentionally unused`, `// reserved`
- Auto-generated files (`.pb.go`, `_generated.`, `moc_`, etc.)

---

## Global Rules

- One dead code unit per commit (one function, one variable, one import, one block).
- After removal: clean up orphaned blank lines (max 1 consecutive blank line after removal).
- After removal: remove any orphaned includes that were only needed by the removed code.
- Do not restructure, rename, or refactor — removal only.
- Do not add comments explaining the removal.

---

## Commit Rules

- Type: always `chore`
- Format: `chore: remove <what> in <location>`
- **Forbidden words:** `cleanup`, `misc`, `various`, `improvements`, `update`

---

## Each Iteration

### Step 1 — Scan

Use static analysis tools if available (priority order):
- C/C++: `cppcheck`, `clang-tidy --checks=clang-analyzer-deadcode.*`
- Go: `go vet`, `staticcheck`
- Python: `pylint --disable=all --enable=W0611,W0612`
- Fallback: model performs static analysis

Output: candidate list (max 5), filtered by `CERTAINTY`.

### Step 2 — Select

Priority: highest certainty → smallest scope → least risky location.
No candidates → `SKIPPED`. Three consecutive SKIPPED → halt, exit code `2`.

### Step 3 — Plan

Output before any change: what, where, risk level, cleanup needed.

### Step 4 — Remove

Remove exactly the plan. Clean orphaned blank lines and includes. Nothing else.

### Step 5 — Verify

- On failure: `git checkout -- <file>`, mark `FALSE-POSITIVE`, continue to next candidate.
- False-positive does NOT increment `fail_count`.
- Only increment `fail_count` for non-removal build failures.
- `fail_count >= STOP_ON_CONSECUTIVE_FAILS` → halt, exit code `1`.

### Step 6 — Commit & Push

```bash
git add .
git commit -m "chore: remove <what> in <location>"
git push
```

Iteration NOT complete until `git push` returns exit code `0`.

---

## End Conditions & Exit Codes

| Condition | Exit Code |
|---|---|
| All iterations completed successfully | `0` |
| Stopped: consecutive fail limit reached | `1` |
| Stopped: no candidates (skip limit) | `2` |
| Stopped: build broken before skill ran | `3` |
| Stopped: invalid configuration | `4` |

---

## Output Formats

### human
```
ITERATION i/N
Category:    <type>
Target:      <file>:<line> — <symbol>
Certainty:   safe | caution
Plan:        <description>
Removed:     <lines> lines
Cleanup:     <orphaned items>
Verify:      <command> → PASS | FAIL | FALSE-POSITIVE
Push:        success | failed
Commit:      chore: remove <what> in <location>
```

### compact
```
[i/N] <category> | <file>:<symbol> -Xln | PASS|FAIL|FP | PUSH:OK|FAIL | chore: <desc>
```

### json
```json
{
  "iteration": 1, "total": 10,
  "category": "unused-function",
  "target": { "file": "auth/token.go", "line": 45, "symbol": "validateLegacy" },
  "certainty": "safe", "status": "success",
  "lines_removed": 12,
  "cleanup": ["removed 2 orphaned blank lines"],
  "validation": { "command": "go build ./...", "result": "pass" },
  "push": "success",
  "commit": "chore: remove unused validateLegacy in auth/token.go",
  "commit_hash": "b2e41fc"
}
```
