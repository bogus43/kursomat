---
name: test-coverage-push
description: >
  Systematically increases test coverage — one function per iteration.
  Reads coverage report, selects lowest-covered function, writes a test,
  verifies coverage increased, commits and pushes.
  Use when asked to: increase test coverage, add missing tests, push coverage
  toward a target, write unit tests for uncovered functions.
  Do NOT use for: fixing bugs, refactoring, integration tests.
  Invoke explicitly with $test-coverage-push or implicitly when prompt matches.
---

# SKILL: TEST COVERAGE PUSH

Senior engineer writing unit tests to increase code coverage. One function per iteration.

---

## Parameters

| Parameter             | Default               | Description                                          |
|----------------------|-----------------------|------------------------------------------------------|
| `ITERATIONS`         | `10`                  | Number of iterations                                 |
| `TARGET_COVERAGE`    | `80`                  | Stop at this % (0 = run all)                         |
| `SCOPE`              | `""`                  | Module/dir; empty = whole repo                       |
| `FRAMEWORK`          | `auto`                | `auto` \| `gtest` \| `catch2` \| `unity` \| `go-test` |
| `MIN_COVERAGE_GAIN`  | `0.1`                 | Min % gain required to commit                        |
| `OUTPUT_FORMAT`      | `human`               | `human` \| `compact` \| `json`                       |
| `LOG_FILE`           | `.coverage-push.log`  | Append log                                           |
| `PUSH_AFTER_COMMIT`  | `true`                | Mandatory push after commit                          |
| `PUSH_FAILURE_ACTION`| `halt`                | `halt` \| `retry`                                    |
| `FORBIDDEN`          | `[deps,vendor,generated,test*]` | Never touch                             |

---

## Framework Detection

| Framework | Signals |
|---|---|
| GTest   | `<gtest/gtest.h>`, `TEST(`, `TEST_F(` |
| Catch2  | `<catch2/catch.hpp>`, `TEST_CASE(` |
| Unity   | `"unity.h"`, `RUN_TEST(`, `TEST_ASSERT_` |
| go-test | `package *_test`, `func Test*(t *testing.T)` |

---

## Each Iteration

1. **Read coverage** — parse report, build ranked candidate list (lowest % first)
2. **Select** — one function; skip constructors, getters, test files, FORBIDDEN
3. **Plan** — paths to cover, expected gain, target test file
4. **Write test** — framework conventions, no production code changes, real behavior
5. **Verify** — all tests pass AND coverage gain ≥ MIN_COVERAGE_GAIN
   - Gain insufficient → `INSUFFICIENT_GAIN`, try next candidate
   - Tests fail → restore, `FAILED`, increment fail_count
6. **Commit & push** — `test: add coverage for <Fn> in <module>`

---

## Commit Format

`test: add coverage for <FunctionName> in <module>`

---

## Exit Codes

| Code | Condition |
|---|---|
| `0` | All iterations done or TARGET_COVERAGE reached |
| `1` | Consecutive fail limit (2) |
| `2` | No candidates (skip limit 3) |
| `3` | Build broken before skill ran |
| `4` | Invalid configuration |
