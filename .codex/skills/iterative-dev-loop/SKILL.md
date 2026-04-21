---
name: iterative-dev-loop
description: >
  Runs disciplined, multi-iteration software development cycles on a git repository.
  Each iteration is a complete mini-cycle: analysis → plan → implementation → verification → commit + push.
  Use when asked to: run iterative fixes, apply a series of commits, run N development iterations,
  execute repeated refactoring passes, or automate disciplined code changes with git commits.
  Do NOT use for: single one-shot edits, code review only, or tasks that don't require git commits.
  Invoke explicitly with $iterative-dev-loop or implicitly when task matches iterative dev workflow.
---

# SKILL: ITERATIVE DEV LOOP

You are a senior engineer running a disciplined, multi-iteration development cycle.
One goal per iteration. One commit per iteration. No broken builds.

---

## Startup Sequence

1. Check for `.iterdev.yml` in repo root — load if present.
2. Check for `.iterdev-context.md` in repo root — load if present (highest priority).
3. Check for inline `CONTEXT:` block in the user prompt — apply as highest priority.
4. Apply defaults for any remaining unset parameters.
5. Never wait for interactive input — start immediately.

> **Priority order:** Inline context → `.iterdev-context.md` → `.iterdev.yml` → defaults

---

## Parameters

| Parameter                       | Default        | Description                                               |
|--------------------------------|----------------|-----------------------------------------------------------|
| `ITERATIONS`                   | `10`           | Number of iterations to run                               |
| `ALLOWED_TYPES`                | `["fix"]`      | List of allowed commit types                              |
| `VALIDATION`                   | `compile`      | `compile` = zero errors; `test` = full test suite         |
| `MAX_FILES_CHANGED`            | `8`            | Hard limit: files changed per iteration                   |
| `MAX_LINES_CHANGED`            | `250`          | Hard limit: lines changed per iteration                   |
| `STRICT_MODE`                  | `true`         | `false` = medium risk → warning only, not halt            |
| `ALLOW_MINOR_REFACTOR_IN_FIX`  | `false`        | Allow trivial incidental cleanup in fix iterations        |
| `REQUIRE_FULL_TEST_PASS`       | `true`         | `false` = no regressions allowed (not 100% required)      |
| `STOP_ON_CONSECUTIVE_FAILS`    | `2`            | Halt after N consecutive build/test failures              |
| `OUTPUT_FORMAT`                | `human`        | `human` \| `compact` \| `json`                           |
| `LOG_FILE`                     | `.iterdev.log` | Append-mode log path; `""` to disable                    |
| `PUSH_AFTER_COMMIT`            | `true`         | `git push` mandatory after every commit                   |
| `PUSH_FAILURE_ACTION`          | `halt`         | `halt` \| `retry` on push failure                        |
| `FORBIDDEN`                    | `["deps"]`     | Paths/patterns never to touch                             |

---

## Global Rules

- Work only inside the current repository.
- Never touch files listed in `FORBIDDEN`.
- Follow existing project conventions — naming, formatting, structure.
- No formatting-only commits unless `ALLOWED_TYPES` includes `style`.
- Changes exceeding `MAX_FILES_CHANGED` or `MAX_LINES_CHANGED`: reduce scope, never exceed.
- `ALLOW_MINOR_REFACTOR_IN_FIX=false` → zero unrelated changes, even trivial ones.

---

## Commit Rules

- Format: `<type>: <short technical description>`
- Type must be from `ALLOWED_TYPES`.
- **Forbidden words:** `update`, `improvements`, `changes`, `misc`, `various`, `minor`.
- No iteration numbers in commit messages.
- One logical change per commit.

---

## Each Iteration

### Step 1 — Analysis

- Scan for candidates matching `ALLOWED_TYPES`.
- Output: up to 3 candidates with file paths and rationale.
- Priority: **lowest risk first, then highest impact**.
- Select exactly one candidate.
- No candidates → mark `SKIPPED`, increment skip counter.
- `skip_count >= 3` consecutively → halt, exit code `2`.

### Step 2 — Action Plan

Output before any code is written:
- Scope: what changes / what does not change
- File list
- Validation command(s)
- Risk level:
  - `low` → proceed
  - `medium` → warn; halt if `STRICT_MODE=true`
  - `high` → halt, request confirmation

### Step 3 — Implementation

- Implement exactly the plan. Nothing extra.
- Scope exceeds limits → reduce scope, re-plan. Never skip the check.

### Step 4 — Verification

- Run `VALIDATION`.
- `compile`: zero errors required.
- `test`: 100% pass (`REQUIRE_FULL_TEST_PASS=true`) or no regressions (`false`).
- On failure: one fix attempt within this iteration.
- Still failed → mark `FAILED`, increment `fail_count`.
- `fail_count >= STOP_ON_CONSECUTIVE_FAILS` → halt, exit code `1`.

### Step 5 — Commit & Push

```bash
git add .
git commit -m "<type>: <description>"
git push
```

- Iteration is **NOT complete** until `git push` returns exit code `0`.
- Skip commit if `NO_EMPTY_COMMITS=true` and nothing staged.
- Reset `fail_count` to `0` after successful push.
- Push failure + `PUSH_FAILURE_ACTION=halt` → halt, report error.
- Push failure + `PUSH_FAILURE_ACTION=retry` → retry once, then halt.

---

## End Conditions & Exit Codes

| Condition                                    | Exit Code |
|----------------------------------------------|-----------|
| All iterations completed successfully        | `0`       |
| Stopped: consecutive fail limit reached      | `1`       |
| Stopped: no valid candidates (skip limit)    | `2`       |
| Stopped: validation broken at session start  | `3`       |
| Stopped: invalid configuration               | `4`       |

- Exit `1`: report all failure causes + last good commit hash.
- Exit `2`: report which types were searched and why no candidates found.

---

## Output Formats

### human
```
ITERATION i/N
Type:         <type>
Problem:      <description>
Plan:         <description>
Files:        <list>
Verification: <command> → PASS | FAIL
Push:         success | failed
Commit:       <type>: <description>
```

### compact
```
[i/N] <type> | <file> +X/-Y | PASS|FAIL | PUSH:OK|FAIL | <type>: <description>
```

### json
```json
{
  "iteration": 1, "total": 10, "type": "fix", "status": "success",
  "problem": "...", "plan": "...", "files": ["..."],
  "lines_added": 12, "lines_removed": 3,
  "validation": { "command": "...", "result": "pass" },
  "push": "success",
  "commit": "fix: ...", "commit_hash": "a3f92bc"
}
```

**Logging:** terminal = `OUTPUT_FORMAT`; log file = always `human`, append mode.

---

## Runtime Context

Two forms accepted — both loaded before iteration 1, highest priority:

**Inline** (quick, single-session):
```
CONTEXT:
Focus on src/i2c/ only
Priority: fix errors at I2C address 0x50
MAX_FILES_CHANGED: 3
```

**File** (reusable, complex tasks) — place `.iterdev-context.md` in repo root:
```
CONTEXT FILE: .iterdev-context.md
```

---

## Allowed Types

| Type       | Purpose                              |
|------------|--------------------------------------|
| `fix`      | Bug fix, no new features             |
| `feat`     | New feature or capability            |
| `refactor` | Code restructure, no behavior change |
| `docs`     | Documentation only                   |
| `test`     | Tests only                           |
| `perf`     | Performance improvement              |
| `security` | Security fix or hardening            |
| `chore`    | Maintenance, tooling, non-code       |
| `build`    | Build system changes                 |
| `ci`       | CI/CD pipeline changes               |
| `style`    | Formatting, whitespace only          |
