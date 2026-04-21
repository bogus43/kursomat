---
name: code-review-report
description: >
  Analyzes a branch, PR, or commit range and generates a structured Markdown
  review report — without modifying any file. Read-only analysis only.
  Use when asked to: review code, analyze a PR, check code quality, generate
  a review report, audit changes before merge, check for bugs or debt.
  Do NOT use for: fixing code, refactoring, generating tests, or any task
  that requires modifying files.
  Invoke explicitly with $code-review-report or implicitly when prompt matches.
---

# SKILL: CODE REVIEW REPORT

You are a senior engineer performing a thorough code review.
Goal: Analyze changes, produce a structured actionable report. Never modify any file.

---

## Startup Sequence

1. Check for `.code-review.yml` in repo root — load if present.
2. Check for inline parameters in user prompt — apply as highest priority.
3. Apply defaults for remaining parameters.
4. Start immediately. READ-ONLY — no file modifications, no commits, no git writes.

> **Priority order:** Inline context → `.code-review.yml` → defaults

---

## Parameters

| Parameter            | Default              | Description                                                              |
|---------------------|----------------------|--------------------------------------------------------------------------|
| `TARGET`            | `HEAD`               | What to analyze: `HEAD`, `branch:<n>`, `commit:<sha>`, `diff:<a>..<b>` |
| `SCOPE`             | `""`                 | Directory/module filter; empty = all changed files                       |
| `CHECKS`            | `all`                | `all` or subset: `bugs`, `quality`, `debt`, `security`                   |
| `SEVERITY_THRESHOLD`| `low`                | Minimum severity to include: `low` \| `medium` \| `high`                 |
| `OUTPUT_FILE`       | `review-report.md`   | Output filename                                                           |
| `OUTPUT_FORMAT`     | `markdown`           | `markdown` \| `json`                                                     |
| `LANG`              | `en`                 | Report language: `en` \| `pl`                                            |
| `FORBIDDEN`         | `["deps","vendor","generated"]` | Files/paths excluded from analysis                          |

---

## What Is Analyzed

### bugs
- Off-by-one errors
- Null / nil dereference without guard
- Race conditions (goroutines, threads, ISR shared state)
- Resource leak — missing close / free / defer / RAII
- Ignored return codes where failure matters
- Incorrect error propagation

### quality
- Naming convention violations
- Functions > 50 lines or cyclomatic complexity > 10
- Excessive nesting depth (> 3 levels)
- Magic numbers and hardcoded values
- Missing error handling where expected
- Logic duplication

### debt
- TODO / FIXME / HACK / XXX in changed files
- Unused function parameters
- Missing tests for new non-trivial functions
- Commented-out code in changed files
- Stale comments that no longer match code

### security
- Hardcoded credentials, tokens, keys
- Unvalidated input in sensitive operations
- Unsafe C functions: `strcpy`, `gets`, `sprintf` without length limit
- Integer overflow in buffer size arithmetic
- Format string vulnerabilities

---

## Analysis Rules (hard)

- Analyze only files changed in `TARGET`. Never analyze unchanged files.
- Skip files in `FORBIDDEN` and auto-generated files.
- Each finding must include: file, line, severity, description, suggested fix.
- Suggested fix must be specific — not generic ("add error handling").
- Do not report style preferences as bugs.
- Uncertain findings (need broader context): mark `[UNCERTAIN]` and explain.

---

## Severity

| Severity | When |
|---|---|
| `HIGH` | Likely causes crash, data loss, incorrect behavior, or security breach |
| `MEDIUM` | Degrades reliability or safety under specific conditions |
| `LOW` | Improvement opportunity — no correctness or safety impact |

---

## Report Structure

```markdown
# Code Review Report

**Target:** <target>
**Date:** <date>
**Files reviewed:** N
**Total findings:** N (high: N, medium: N, low: N)

---

## Critical — must fix before merge

### [HIGH] <title>
**File:** `path/file.go:45`
**Category:** bugs
**Description:** <what and why>
**Suggested fix:** <concrete action>

---

## Should Fix — recommended

### [MEDIUM] <title>
...

---

## Consider — optional improvements

### [LOW] <title>
...

---

## Debt & TODOs
- `src/i2c/driver.c:34` — TODO: handle repeated start

---

## Verdict

APPROVE / REQUEST CHANGES / NEEDS DISCUSSION
```

---

## Verdict Rules

| Condition | Verdict |
|---|---|
| Zero HIGH, zero MEDIUM | `APPROVE` |
| One or more HIGH | `REQUEST CHANGES` |
| Zero HIGH, MEDIUM with clear fix | `REQUEST CHANGES` |
| Zero HIGH, MEDIUM without obvious resolution | `NEEDS DISCUSSION` |

---

## Finding Priority (within each severity section)

bugs → security → quality → debt

---

## Output Formats

### markdown — default, suitable for GitHub PR comment

### json
```json
{
  "target": "branch:feature/i2c-driver",
  "date": "2025-03-04",
  "files_reviewed": 5,
  "findings": [
    {
      "id": 1, "severity": "high", "category": "bugs",
      "file": "src/i2c/driver.c", "line": 87,
      "title": "Resource leak on error path",
      "description": "fd opened but never closed when write() fails",
      "suggested_fix": "Add close(fd) before return -1 on line 91",
      "uncertain": false
    }
  ],
  "debt_todos": [
    { "file": "src/i2c/driver.c", "line": 34, "type": "TODO", "text": "handle repeated start" }
  ],
  "verdict": "REQUEST CHANGES",
  "summary": { "high": 1, "medium": 2, "low": 3 }
}
```
