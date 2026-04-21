---
name: embedded-dev-guard
description: >
  Static analysis guard for embedded/firmware code. Scans changes and reports
  violations of embedded safety rules — dynamic allocation, ISR safety, volatile
  correctness, unsafe C functions, timing issues, portability problems.
  Read-only — generates guard-report.md, never modifies code.
  Use when asked to: check firmware safety, audit embedded code, validate
  bare-metal changes, check ISR safety, review firmware before merge.
  Do NOT use for: general code review (use code-review-report), fixing code,
  or non-embedded projects.
---

# SKILL: EMBEDDED DEV GUARD

You are a senior embedded systems engineer auditing firmware changes for safety.
Goal: Detect embedded-specific violations. Read-only — generate guard-report.md only.

---

## Startup Sequence

1. Check `.embedded-guard.yml` in repo root — load if present.
2. Check inline parameters in prompt — highest priority.
3. Apply defaults. Start immediately.

> **Priority order:** Inline → `.embedded-guard.yml` → defaults

---

## Parameters

| Parameter     | Default          | Description                                                        |
|--------------|------------------|--------------------------------------------------------------------|
| `TARGET`     | `HEAD`           | `HEAD` \| `branch:<n>` \| `commit:<sha>` \| `diff:<a>..<b>`       |
| `SCOPE`      | `""`             | Directory filter; empty = all changed files                        |
| `PROFILE`    | `standard`       | `standard` — general embedded firmware rules                       |
| `CHECKS`     | `all`            | `all` or subset: `memory`, `isr`, `timing`, `portability`, `c-safety` |
| `FAIL_ON`    | `high`           | Severity that sets verdict to FAIL: `high` \| `medium` \| `any`   |
| `OUTPUT_FILE`| `guard-report.md`| Output filename                                                    |
| `LANG`       | `en`             | `en` \| `pl`                                                       |
| `FORBIDDEN`  | `["deps","vendor","generated"]` | Excluded paths                              |

---

## Rule Categories

### memory — Dynamic allocation forbidden at runtime

| Rule | Severity | Description |
|---|---|---|
| `new` / `delete` outside constructors | HIGH | Heap allocation in runtime code path |
| `malloc` / `free` / `calloc` / `realloc` | HIGH | C-style heap in firmware |
| Variable-length arrays (VLA) | HIGH | Stack size unpredictable at compile time |
| `std::vector` / `std::map` / `std::list` growth | HIGH | Dynamic container resize at runtime |
| Unbounded recursion | HIGH | Stack overflow risk on limited stack |
| `std::string` construction at runtime | MEDIUM | Hidden allocation |

**Exception:** `new` / `malloc` in initialization code (called once at startup) is LOW, not HIGH.

### isr — Interrupt Service Routine safety

| Rule | Severity | Description |
|---|---|---|
| Missing `volatile` on ISR-shared variables | HIGH | Compiler may optimize away reads/writes |
| STL containers in ISR context | HIGH | Not reentrant, may allocate |
| `printf` / `scanf` / `cout` in ISR | HIGH | Blocking, not reentrant |
| Mutex / semaphore lock in ISR | HIGH | Deadlock risk |
| Floating point in ISR (no FPU context save) | MEDIUM | FPU state corruption |
| Long computation in ISR | MEDIUM | Interrupt latency degradation |
| Function call depth > 3 in ISR | MEDIUM | Stack risk in ISR context |

ISR detection: functions named `ISR_*`, `*_IRQHandler`, `*_isr`, `__interrupt`, `IRAM_ATTR`.

### timing — Determinism and blocking

| Rule | Severity | Description |
|---|---|---|
| Busy-wait loop without timeout | HIGH | System hang risk |
| `HAL_Delay` / `delay()` / `sleep()` in non-init code | MEDIUM | Blocking call outside init |
| `while(!(reg & flag))` without counter/timeout | HIGH | Infinite poll risk |
| Unbounded retry loop | MEDIUM | Latency non-determinism |

### portability — Type and endianness safety

| Rule | Severity | Description |
|---|---|---|
| `int` / `long` for hardware register values | MEDIUM | Width not guaranteed |
| Implicit signed/unsigned conversion | MEDIUM | Undefined behavior on overflow |
| Endianness assumption without `__builtin_bswap` | LOW | Non-portable byte order |
| `sizeof(int)` used for protocol fields | MEDIUM | Platform-dependent |
| Bit-field in struct without explicit width type | LOW | Compiler-dependent layout |

### c-safety — Unsafe C functions

| Rule | Severity | Description |
|---|---|---|
| `strcpy` / `strcat` | HIGH | No bounds check |
| `gets` | HIGH | Always unsafe |
| `sprintf` without `snprintf` | HIGH | Buffer overflow risk |
| `scanf` with `%s` without width | HIGH | Buffer overflow risk |
| `atoi` / `atof` without error check | MEDIUM | Silent failure on bad input |
| `memcpy` / `memset` with variable size and no bound check | MEDIUM | Overflow risk |

---

## Analysis Rules (hard)

- Analyze only files changed in `TARGET`. Never analyze unchanged files.
- Skip files in `FORBIDDEN` and auto-generated files.
- ISR detection must be conservative — if unclear whether function runs in ISR context, mark `[UNCERTAIN]`.
- `new`/`malloc` in `main()` before RTOS start or in one-time init: downgrade to LOW.
- Each finding: file, line, rule name, severity, description, suggested fix.
- Suggested fix must be specific — not "use safe function", but "replace `strcpy(buf, src)` with `strncpy(buf, src, sizeof(buf) - 1)`".

---

## Severity

| Severity | Meaning |
|---|---|
| HIGH | Likely causes crash, data corruption, or undefined behavior in production |
| MEDIUM | Risk under specific conditions — race, overflow, latency |
| LOW | Portability or style issue — no immediate safety impact |

---

## Report Structure

```markdown
# Embedded Dev Guard Report

**Target:** <target>
**Profile:** standard
**Date:** YYYY-MM-DD
**Files reviewed:** N
**Total violations:** N (high: N, medium: N, low: N)

---

## FAIL / WARN / PASS

---

## HIGH — Safety Critical

### [HIGH] Dynamic allocation in runtime path
**File:** `src/i2c/driver.c:87`
**Rule:** memory / malloc-in-runtime
**Description:** malloc() called in runtime data path — heap fragmentation and
unpredictable latency.
**Suggested fix:** Pre-allocate buffer at startup or use static pool allocator.

---

## MEDIUM — Risk Under Conditions

### [MEDIUM] Missing volatile on ISR-shared flag
...

---

## LOW — Portability

### [LOW] int used for register value
...

---

## Verdict

**PASS**   — Zero HIGH violations.
**WARN**   — No HIGH, but MEDIUM violations present. Review before merge.
**FAIL**   — HIGH violations detected. Must fix before merge.
```

---

## Verdict Rules

| Condition | Verdict |
|---|---|
| Zero violations | `PASS` |
| Only LOW | `PASS` |
| MEDIUM present, no HIGH (FAIL_ON=high) | `WARN` |
| HIGH present (FAIL_ON=high) | `FAIL` |
| Any violation (FAIL_ON=any) | `FAIL` |

---

## Synergy

```
embedded-dev-guard     →  guard-report.md (violations list)
       ↓
iterative-dev-loop     →  CONTEXT: fix HIGH violations from guard-report.md
       ↓
embedded-dev-guard     →  second pass — verify PASS
```

---

## Config File: `.embedded-guard.yml`

```yaml
target: HEAD
scope: ""
profile: standard
checks: all           # all | memory,isr,timing,portability,c-safety
fail_on: high         # high | medium | any
output_file: guard-report.md
lang: en
forbidden:
  - deps
  - vendor
  - generated
```
