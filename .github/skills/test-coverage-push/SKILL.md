---
name: test-coverage-push
description: >
  Systematically increases test coverage — one function per iteration.
  Reads coverage report, selects lowest-covered function, writes a test,
  verifies coverage increased, commits and pushes.
  Use when asked to: increase test coverage, add missing tests, push coverage
  toward a target, write unit tests for uncovered functions.
  Do NOT use for: fixing bugs, refactoring, or writing integration tests.
---

# SKILL: TEST COVERAGE PUSH

You are a senior engineer writing unit tests to increase code coverage.
Goal: One function per iteration — read coverage, pick target, write test, verify gain, commit.

---

## Startup Sequence

1. Check `.test-coverage.yml` in repo root — load if present.
2. Check inline parameters in prompt — highest priority.
3. Apply defaults. Start immediately.

> **Priority order:** Inline → `.test-coverage.yml` → defaults

---

## Parameters

| Parameter            | Default      | Description                                                         |
|---------------------|--------------|---------------------------------------------------------------------|
| `ITERATIONS`        | `10`         | Number of iterations                                                |
| `TARGET_COVERAGE`   | `80`         | Stop when overall coverage reaches this % (0 = run all iterations) |
| `SCOPE`             | `""`         | Module/directory; empty = whole repo                                |
| `FRAMEWORK`         | `auto`       | `auto` \| `gtest` \| `catch2` \| `unity` \| `go-test`             |
| `MIN_COVERAGE_GAIN` | `0.1`        | Minimum % gain for commit to be valid                               |
| `FORBIDDEN`         | `["deps","vendor","generated","test","tests","*_test*","*_spec*"]` | Never touch |
| `OUTPUT_FORMAT`     | `human`      | `human` \| `compact` \| `json`                                     |
| `LOG_FILE`          | `.coverage-push.log` | Append log; `""` to disable                               |
| `PUSH_AFTER_COMMIT` | `true`       | git push mandatory after every commit                               |
| `PUSH_FAILURE_ACTION`| `halt`      | `halt` \| `retry`                                                   |

---

## Framework Detection (FRAMEWORK=auto)

| Framework | Detection signals |
|---|---|
| GTest | `#include <gtest/gtest.h>`, `CMakeLists.txt` with `gtest`, `TEST(`, `TEST_F(` |
| Catch2 | `#include <catch2/catch.hpp>`, `TEST_CASE(`, `REQUIRE(` |
| Unity | `#include "unity.h"`, `RUN_TEST(`, `TEST_ASSERT_` |
| go-test | `package *_test`, `func Test*(t *testing.T)`, `go test` |

If multiple: use framework already present in nearest test file to target.

---

## Coverage Report Detection

Auto-detect coverage format:

| Tool | File | Command to generate |
|---|---|---|
| lcov (C/C++) | `coverage.info` / `lcov.info` | `lcov --capture --directory . --output-file coverage.info` |
| gcov (C/C++) | `*.gcov` files | `gcov <source-file>` |
| go test | inline | `go test ./... -coverprofile=coverage.out` |
| llvm-cov | `coverage.json` | `llvm-cov export -format=text` |

If no coverage report found: run coverage command first (detect toolchain from Makefile/CMake/go.mod), then proceed.

---

## Candidate Selection

From coverage report, build ranked list:

1. Filter to `SCOPE` if set
2. Filter out files in `FORBIDDEN`
3. Filter out test files (`*_test.go`, `test_*.c`, `*_test.cpp`, `*Spec.*`)
4. Filter out auto-generated (`*.pb.go`, `_generated.`, `moc_`)
5. Sort by function coverage % ascending
6. Within same coverage %: prefer smaller functions (fewer lines — easier to test)
7. Prefer functions with existing tests nearby (test file already exists)
8. Skip: constructors, destructors, pure getters/setters with zero logic

Select exactly one function per iteration.

No candidates → `SKIPPED`. Three consecutive SKIPPED → halt, exit code `2`.

---

## Test Writing Rules

- Write minimal, focused test — one logical scenario per test case
- Test name must describe what is being tested: `TEST(AuthModule, ValidateToken_ReturnsErrorOnExpiry)`
- Use existing test fixtures/helpers if present in the test file
- Do not test implementation details — test observable behavior
- Public API + internal functions both in scope
- For functions with multiple paths: start with the happy path, add error path in same test if coverage gain would be insufficient otherwise
- Add test to existing test file for the module if one exists — do not create unnecessary new files
- If no test file exists: create `test_<module>.<ext>` or `<module>_test.<ext>` following project convention

**Never:**
- Modify production code to make tests pass
- Write tests that always pass regardless of implementation (vacuous tests)
- Mock everything — test real behavior where possible

---

## Each Iteration

### Step 1 — Read Coverage
Parse coverage report. Build ranked candidate list.
If report outdated (older than source files): regenerate first.

### Step 2 — Select
Pick lowest-coverage function meeting filter criteria.
Report current coverage % before change.

### Step 3 — Plan
- Function name, file, current coverage %
- Which paths will be covered by new test
- Expected coverage gain (estimate)
- Test file target (existing or new)

### Step 4 — Write Test
Write test following framework conventions. Do not modify production code.

### Step 5 — Verify
```bash
# Run tests + regenerate coverage
# Then compare new coverage % to old
```

Check two conditions:
1. All tests pass (including pre-existing)
2. Coverage increased by at least `MIN_COVERAGE_GAIN`%

If tests pass but coverage gain < `MIN_COVERAGE_GAIN`: mark `INSUFFICIENT_GAIN` — do not commit, try next candidate.
If tests fail: restore test file, mark `FAILED`, increment fail_count.
If `fail_count >= 2`: halt, exit code `1`.

### Step 6 — Commit & Push
```bash
git add <test file>
git commit -m "test: add coverage for <FunctionName> in <module>"
git push
```

Iteration NOT complete until push returns exit code `0`.
Reset fail_count after successful push.
Report new coverage %.

---

## Stop Conditions

| Condition | Exit Code |
|---|---|
| All iterations completed | `0` |
| TARGET_COVERAGE reached | `0` (report: "Target N% reached") |
| Consecutive fail limit | `1` |
| No candidates (skip limit) | `2` |
| Build broken before skill ran | `3` |
| Invalid configuration | `4` |

---

## Commit Rules

- Type: always `test`
- Format: `test: add coverage for <FunctionName> in <module>`
- Examples:
  - `test: add coverage for ValidateToken in auth/token.go`
  - `test: add coverage for i2c_write_frame in src/i2c/driver.c`
- Forbidden words: `improve`, `increase`, `more`, `better`, `misc`

---

## Output Formats

### human
```
ITERATION 1/10
Target:     auth/token.go — ValidateToken (current: 23%)
Framework:  go-test
Plan:       Test happy path + expired token error path
Test file:  auth/token_test.go (existing)
Verify:     go test ./auth/... → PASS | coverage: 23% → 41% (+18%)
Push:       success
Commit:     test: add coverage for ValidateToken in auth/token.go
Coverage:   project 61% → 63%
```

### compact
```
[1/10] auth/token.go:ValidateToken 23%→41% +18% | PASS | PUSH:OK | test: add coverage for ValidateToken
[2/10] src/i2c/driver.c:i2c_read 0%→45% +45%   | PASS | PUSH:OK | test: add coverage for i2c_read
[3/10] INSUFFICIENT_GAIN | db/conn.go:ping +0.0% | trying next candidate
```

---

## Config File: `.test-coverage.yml`

```yaml
iterations: 10
target_coverage: 80       # 0 = run all iterations regardless
# scope: ""               # empty = whole repo; e.g. "src/i2c/"
framework: auto           # auto | gtest | catch2 | unity | go-test
min_coverage_gain: 0.1
output_format: human      # human | compact | json
log_file: .coverage-push.log
push_after_commit: true
push_failure_action: halt
forbidden:
  - deps
  - vendor
  - generated
```
