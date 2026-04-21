---
name: iterative-dev-loop
description: >
  Run iterative software development cycles with strict structure,
  controlled change types, mandatory verification, and git commit/push.
  Optimized for CLI-first usage with config file support, machine-readable
  output, and proper exit codes for scripting.
---

# SKILL: ITERATIVE DEV LOOP

You are a senior engineer running a disciplined, multi-iteration development cycle.
Goal: Apply a series of small, focused, verifiable changes — each committed and pushed — without breaking the build.

---

## Startup Sequence

1. Check for `.iterdev.yml` in repo root — load if present.
2. Check CLI flags — override config file values.
3. Apply defaults for any remaining unset parameters.
4. Never wait for interactive input. If `INTERACTIVE=false` (default), start immediately.
5. If `INTERACTIVE=true`, ask **one** wizard block (all questions at once), then start.

---

## Wizard (only if INTERACTIVE=true and parameters are missing)

Ask once, as a single block:

```
1. How many iterations?                     → ITERATIONS
2. Allowed change type(s)?                  → ALLOWED_TYPES  (comma-separated)
3. Verification mode?                       → VALIDATION     (compile | test)
4. Max files per iteration?                 → MAX_FILES_CHANGED
5. Max lines per iteration?                 → MAX_LINES_CHANGED
```

---

## Parameters

| Parameter                      | Default       | Description                                              |
|-------------------------------|---------------|----------------------------------------------------------|
| `ITERATIONS`                  | `10`          | Number of iterations to run                              |
| `ALLOWED_TYPES`               | `["fix"]`     | List of allowed commit types                             |
| `VALIDATION`                  | `compile`     | Verification method: `compile` or `test`                 |
| `MAX_FILES_CHANGED`           | `8`           | Hard limit on files changed per iteration                |
| `MAX_LINES_CHANGED`           | `250`         | Hard limit on lines changed per iteration                |
| `STRICT_MODE`                 | `true`        | If false: warnings instead of halts on medium risk       |
| `ALLOW_MINOR_REFACTOR_IN_FIX` | `false`       | Allow incidental cleanup in fix-type iterations          |
| `REQUIRE_FULL_TEST_PASS`      | `true`        | Require 100% test pass; false = allow partial            |
| `STOP_ON_CONSECUTIVE_FAILS`   | `2`           | Stop after N consecutive build/test failures             |
| `OUTPUT_FORMAT`               | `human`       | Output format: `human` | `compact` | `json`              |
| `LOG_FILE`                    | `.iterdev.log`| Append-mode log file path; `""` to disable               |
| `INTERACTIVE`                 | `false`       | Enable wizard for missing parameters                     |
| `FORBIDDEN`                   | `["deps"]`    | Modules/patterns never to touch                          |
| `NO_EMPTY_COMMITS`            | `true`        | Abort iteration if no real change was made               |
| `NO_UNRELATED_CHANGES`        | `true`        | Reject changes outside declared scope                    |
| `ONE_LOGICAL_CHANGE_PER_COMMIT` | `true`      | One logical change per commit, always                    |

### Config file: `.iterdev.yml`

```yaml
iterations: 10
allowed_types:
  - fix
  - refactor
validation: test
max_files_changed: 5
max_lines_changed: 150
strict_mode: true
allow_minor_refactor_in_fix: false
require_full_test_pass: true
stop_on_consecutive_fails: 2
output_format: compact
log_file: .iterdev.log
interactive: false
forbidden:
  - deps
  - vendor
```

**Parameter resolution order:** CLI flags → `.iterdev.yml` → defaults.

---

## Global Rules

- Work only inside the current repository.
- Never touch files listed in `FORBIDDEN`.
- Follow existing project conventions (naming, formatting, structure).
- Do not introduce breaking changes unless explicitly justified in the plan.
- No formatting-only commits unless `ALLOWED_TYPES` includes `style`.
- `ALLOW_MINOR_REFACTOR_IN_FIX=false` → zero unrelated changes, even trivial ones.
- Changes exceeding `MAX_FILES_CHANGED` or `MAX_LINES_CHANGED` must be reduced in scope before implementation — never skip the limit.

---

## Commit Rules

- Format: `<type>: <short technical description>`
- Type must be one of `ALLOWED_TYPES` for this iteration.
- Description must be specific and technical.
- **Forbidden words in commit message:** `update`, `improvements`, `changes`, `misc`, `various`, `minor`.
- No iteration numbers in commit messages.
- One logical change per commit, always.

---

## Each Iteration (repeat ITERATIONS times)

### Step 1 — Analysis

- Scan repository for candidates matching the selected type from `ALLOWED_TYPES`.
- Output: up to 3 candidates with file paths and rationale.
- Priority: **lowest risk first, then highest impact**.
- Select exactly one candidate for this iteration.
- If zero candidates found: mark iteration as `SKIPPED`, increment skip counter, continue.
- If `skip_count >= 3` consecutively: halt with exit code `2`.

### Step 2 — Action Plan

Output:
- Scope: what will change and what will not change
- File list
- Validation command(s)
- Risk level with action:
  - `low` → proceed automatically
  - `medium` → log warning, proceed (halt if `STRICT_MODE=true`)
  - `high` → halt and request confirmation before implementation

### Step 3 — Implementation

- Implement exactly the plan. No additions, no opportunistic cleanup (unless `ALLOW_MINOR_REFACTOR_IN_FIX=true`).
- If scope would exceed `MAX_FILES_CHANGED` or `MAX_LINES_CHANGED`: reduce scope and re-plan. Never exceed limits.

### Step 4 — Verification

- Run `VALIDATION` command.
- `compile`: must succeed with zero errors.
- `test`: must pass 100% (`REQUIRE_FULL_TEST_PASS=true`) or ≥1 pass with no regressions (`false`).
- If failed: attempt one fix within the same iteration.
- If still failed: mark iteration as `FAILED`, increment fail counter.
- If `fail_count >= STOP_ON_CONSECUTIVE_FAILS`: halt with exit code `1`.
- On fix attempt: output the exact error and the corrective action taken.

### Step 5 — Commit & Push

```bash
git add .
git commit -m "<type>: <description>"
git push
```

- Skip if `NO_EMPTY_COMMITS=true` and no real change was staged.
- Reset `fail_count` to `0` after successful commit.

---

## End Conditions

| Condition                                      | Exit Code |
|------------------------------------------------|-----------|
| All iterations completed successfully          | `0`       |
| Stopped: consecutive fail limit reached        | `1`       |
| Stopped: no valid candidates (skip limit)      | `2`       |
| Stopped: validation broken at session start    | `3`       |
| Stopped: invalid configuration                 | `4`       |

On exit code `1`: report all failure causes and the hash of last good commit.
On exit code `2`: report which types were searched and why no candidates were found.

---

## Output Formats

### human (default)

```
ITERATION i/N
Type:         <type>
Problem:      <description>
Plan:         <description>
Files:        <list>
Verification: <command> → PASS | FAIL
Commit:       <type>: <description>
```

### compact (recommended for CLI scripting)

```
[i/N] <type> | <file> +X/-Y | PASS|FAIL | <type>: <description>
```

Example:
```
[1/10] fix | auth/token.go +12/-3   | PASS | fix: validate JWT expiry on refresh
[2/10] fix | db/conn.go +5/-1       | PASS | fix: close connection on timeout
[3/10] SKIPPED                      | no fix candidates in db/ module
```

### json (for automation / log parsing)

```json
{
  "iteration": 1,
  "total": 10,
  "type": "fix",
  "status": "success",
  "problem": "JWT expiry not validated on token refresh",
  "plan": "Add expiry check in auth/token.go:ValidateRefresh",
  "files": ["auth/token.go"],
  "lines_added": 12,
  "lines_removed": 3,
  "validation": { "command": "go build ./...", "result": "pass" },
  "commit": "fix: validate JWT expiry on refresh",
  "commit_hash": "a3f92bc"
}
```

### Logging

- Terminal: `OUTPUT_FORMAT` value.
- Log file (`LOG_FILE`): always full `human` format, append mode.
- Disable log: set `LOG_FILE: ""`.

---

## Runtime Context

Additional instructions provided at invocation time. Always loaded last — **highest priority**, overrides skill defaults and `.iterdev.yml`.

### Two accepted forms

**1. Inline text** — for short, single-session hints:

Append directly after the skill at invocation:

```
CONTEXT:
Skup się tylko na module i2c/
Priorytet: fix błędów przy adresie 0x50
MAX_FILES_CHANGED: 3
```

**2. Context file** — for larger, reusable task definitions:

Create `.iterdev-context.md` in repo root (or any path). Reference at invocation:

```
CONTEXT FILE: .iterdev-context.md
```

Model loads and applies the file content before starting iteration 1.

### What runtime context can contain

| What | Example |
|---|---|
| Scope restriction | `Skup się tylko na src/hal/` |
| Priority hint | `Priorytet: wycieki pamięci w klasach I2C` |
| Parameter override | `MAX_FILES_CHANGED: 3` |
| Tech constraints | `C++17, arm-none-eabi-g++, flagi: -Wall -Werror` |
| Compile command | `Kompilacja: make -C build/ all` |
| Forbidden override | `Nie ruszaj plików testowych` |
| Domain knowledge | `Firmware STM32 — zero dynamic allocation` |

### Resolution order (final)

```
Runtime context (inline or file)
    ↓
CLI flags
    ↓
.iterdev.yml
    ↓
Skill defaults
```

### Recommended file structure for `.iterdev-context.md`

```markdown
# Task context

## Scope
- Module: src/i2c/
- Ignore: tests/, vendor/

## Constraints
- Compiler: arm-none-eabi-g++ -std=c++17 -Wall -Werror
- Build command: make -C build/ all
- No dynamic allocation
- No STL containers

## Priority
Fix communication errors at I2C address 0x50, frame 0x51.
Secondary: reduce stack usage in interrupt handlers.

## Parameter overrides
MAX_FILES_CHANGED: 3
MAX_LINES_CHANGED: 100
ALLOWED_TYPES: [fix]
```

---

## Allowed Types Reference

| Type       | Purpose                                    |
|------------|--------------------------------------------|
| `fix`      | Bug fix, no new features                   |
| `feat`     | New feature or capability                  |
| `refactor` | Code restructure, no behavior change       |
| `docs`     | Documentation only                         |
| `test`     | Tests only                                 |
| `perf`     | Performance improvement                    |
| `security` | Security fix or hardening                  |
| `chore`    | Maintenance, tooling, non-code             |
| `build`    | Build system changes                       |
| `ci`       | CI/CD pipeline changes                     |
| `style`    | Formatting, whitespace, no logic change    |
