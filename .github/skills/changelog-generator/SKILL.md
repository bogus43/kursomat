---
name: changelog-generator
description: >
  Generates or updates CHANGELOG.md from git commit history using only
  git log — no external tools required. Groups commits by type, suggests
  semver version bump, supports Keep a Changelog format.
  Use when asked to: generate a changelog, update CHANGELOG.md, summarize
  changes since last release, prepare release notes, suggest version bump.
  Do NOT use for: code changes, refactoring, or any task that modifies
  source files other than CHANGELOG.md.
---

# SKILL: CHANGELOG GENERATOR

You are a release engineer generating a structured changelog from git history.
Goal: Produce a clean, human-readable CHANGELOG.md entry with zero external dependencies.

---

## Startup Sequence

1. Check for `.changelog.yml` in repo root — load if present.
2. Check for inline parameters in user prompt — highest priority.
3. Apply defaults.
4. Run `git log` to gather commits — start immediately.

> **Priority order:** Inline context → `.changelog.yml` → defaults

---

## Parameters

| Parameter  | Default  | Description                                                          |
|-----------|----------|----------------------------------------------------------------------|
| `FROM`    | `auto`   | Start ref: tag/SHA; `auto` = last git tag                            |
| `TO`      | `HEAD`   | End ref                                                              |
| `VERSION` | `auto`   | Version string; `auto` = suggest based on semver rules               |
| `DRY_RUN` | `true`   | `true` = print only; `false` = write/append to CHANGELOG.md         |
| `APPEND`  | `true`   | `true` = prepend new entry to existing file; `false` = overwrite     |
| `LANG`    | `en`     | Section header language: `en` \| `pl`                                |

---

## Git Log Command

```bash
git log <FROM>..<TO> --pretty=format:"%H %s" --no-merges
```

If `FROM=auto`:
```bash
git describe --tags --abbrev=0   # get last tag
```

If no tags exist: use entire history.

---

## Commit Categorization

Commits follow conventional commits — map directly:

| Commit prefix | Changelog section |
|---|---|
| `feat:` / `feat(*):`         | Added |
| `fix:` / `fix(*):` / `perf:` | Fixed |
| `refactor:` / `style:`       | Changed |
| `docs:`                      | Changed |
| `chore:` / `build:` / `ci:`  | Changed |
| `test:`                      | Changed |
| `security:`                  | Security |
| `BREAKING CHANGE` in body    | ⚠️ Breaking |

Non-conventional commits: categorize by content — model reads subject line and assigns best-fit section.

**Skip entirely:**
- Merge commits
- Commits matching: `wip:`, `tmp:`, `temp:`, `WIP`
- Auto-generated commits (e.g. from iterative-dev-loop with iteration numbers)

---

## Semver Bump Rules

| Commits present | Suggested bump |
|---|---|
| Any `BREAKING CHANGE` | **major** X.0.0 |
| Any `feat:` (no breaking) | **minor** x.Y.0 |
| Only `fix:` / `perf:` / `chore:` | **patch** x.y.Z |
| Only `docs:` / `style:` / `test:` | **patch** x.y.Z |

If `VERSION=auto`: suggest bump, show current → new version, write suggested version into entry.
If last tag is not semver: suggest `1.0.0` as first release.

---

## Output Format (Keep a Changelog)

```markdown
## [X.Y.Z] — YYYY-MM-DD

### ⚠️ Breaking Changes
- Description of breaking change

### Added
- feat: short description (commit sha short)

### Fixed
- fix: short description

### Changed
- refactor/chore/docs/style entries

### Security
- security entries
```

**Rules:**
- Each entry: one line, starts with `- `
- Strip commit type prefix from description: `feat: add I2C retry` → `Add I2C retry`
- Capitalize first word
- Include short SHA in parentheses: `(a3f92bc)`
- Empty sections: omit entirely
- Order: Breaking → Added → Fixed → Changed → Security

---

## DRY_RUN behavior

`DRY_RUN=true` (default):
- Print generated entry to stdout
- Print semver suggestion
- Do NOT write any file
- Print: `Run with DRY_RUN=false to write to CHANGELOG.md`

`DRY_RUN=false`:
- Prepend entry to CHANGELOG.md (if APPEND=true)
- If CHANGELOG.md does not exist: create with standard header
- Do NOT commit — user decides when to commit

---

## CHANGELOG.md Header (if creating new file)

```markdown
# Changelog

All notable changes to this project will be documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

---
```

---

## Config File: `.changelog.yml`

```yaml
from: auto        # auto = last git tag
to: HEAD
version: auto     # auto = semver suggestion
dry_run: true
append: true
lang: en          # en | pl
```
