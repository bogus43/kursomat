---
name: embedded-dev-guard
description: >
  Static analysis guard for embedded/firmware code. Scans changes and reports
  violations of embedded safety rules — dynamic allocation, ISR safety, volatile
  correctness, unsafe C functions, timing issues, portability problems.
  Read-only — generates guard-report.md, never modifies code.
  Use when asked to: check firmware safety, audit embedded code, validate
  bare-metal changes, check ISR safety, review firmware before merge.
  Do NOT use for: general code review, fixing code, non-embedded projects.
  Invoke explicitly with $embedded-dev-guard or implicitly when prompt matches.
---

# SKILL: EMBEDDED DEV GUARD

Senior embedded engineer auditing firmware for safety violations. Read-only.

---

## Parameters

| Parameter     | Default           | Description                                               |
|--------------|-------------------|-----------------------------------------------------------|
| `TARGET`     | `HEAD`            | `HEAD` \| `branch:<n>` \| `commit:<sha>` \| `diff:<a>..<b>` |
| `SCOPE`      | `""`              | Directory filter                                          |
| `CHECKS`     | `all`             | `all` or subset: `memory,isr,timing,portability,c-safety` |
| `FAIL_ON`    | `high`            | `high` \| `medium` \| `any`                               |
| `OUTPUT_FILE`| `guard-report.md` | Output filename                                           |
| `LANG`       | `en`              | `en` \| `pl`                                              |
| `FORBIDDEN`  | `[deps,vendor,generated]` | Excluded paths                                  |

---

## Rule Summary

**memory:** `new`/`delete`/`malloc`/`free` at runtime (HIGH), VLA (HIGH), unbounded recursion (HIGH), `std::string` runtime construction (MEDIUM)

**isr:** missing `volatile` on shared vars (HIGH), STL/printf/mutex in ISR (HIGH), FP without FPU save (MEDIUM). ISR detection: `ISR_*`, `*_IRQHandler`, `*_isr`, `__interrupt`, `IRAM_ATTR`

**timing:** busy-wait without timeout (HIGH), blocking calls outside init (MEDIUM), infinite poll (HIGH)

**portability:** `int` for register values (MEDIUM), signed/unsigned implicit conversion (MEDIUM), endianness assumption (LOW)

**c-safety:** `strcpy`/`gets`/`sprintf`/`scanf %s` without bounds (HIGH), `atoi` without error check (MEDIUM)

**Exception:** `new`/`malloc` in one-time init / before RTOS start → downgrade to LOW.

---

## Verdict

| Condition | Verdict |
|---|---|
| Zero violations or only LOW | `PASS` |
| MEDIUM, no HIGH (FAIL_ON=high) | `WARN` |
| HIGH present | `FAIL` |

---

## Report Structure

```markdown
# Embedded Dev Guard Report
**Target:** ... **Profile:** standard **Date:** ... **Violations:** N (high:N medium:N low:N)

## FAIL / WARN / PASS

## HIGH — Safety Critical
### [HIGH] <rule name>
**File:** `path:line` **Rule:** category/rule-id
**Description:** what and why
**Suggested fix:** specific action

## MEDIUM — Risk Under Conditions
## LOW — Portability
## Verdict: PASS / WARN / FAIL
```
