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
  Invoke explicitly with $changelog-generator or implicitly when prompt matches.
---

# SKILL: CHANGELOG GENERATOR

You are a release engineer generating a structured changelog from git history.
Goal: Produce a clean, human-readable CHANGELOG.md entry. Zero external dependencies.

---

## Startup Sequence

1. Check `.changelog.yml` in repo root — load if present.
2. Inline parameters in prompt — highest priority.
3. Apply defaults. Start immediately.

> **Priority order:** Inline → `.changelog.yml` → defaults

---

## Parameters

| Parameter  | Default | Description                                                  |
|-----------|---------|--------------------------------------------------------------|
| `FROM`    | `auto`  | Start ref; `auto` = last git tag                             |
| `TO`      | `HEAD`  | End ref                                                      |
| `VERSION` | `auto`  | Version; `auto` = semver suggestion                          |
| `DRY_RUN` | `true`  | `true` = print only; `false` = write CHANGELOG.md           |
| `APPEND`  | `true`  | Prepend to existing file                                     |
| `LANG`    | `en`    | `en` \| `pl`                                                 |

---

## Git Commands

```bash
git describe --tags --abbrev=0          # get last tag (FROM=auto)
git log <FROM>..<TO> --pretty=format:"%H %s" --no-merges
```

---

## Commit Mapping

| Prefix | Section |
|---|---|
| `feat:` | Added |
| `fix:` / `perf:` | Fixed |
| `refactor:` / `style:` / `docs:` / `chore:` / `build:` / `ci:` / `test:` | Changed |
| `security:` | Security |
| `BREAKING CHANGE` in body | ⚠️ Breaking |

Non-conventional: model categorizes by content. Skip: merge commits, `wip:`, `tmp:`, `temp:`.

---

## Semver Rules

| Commits | Bump |
|---|---|
| Any `BREAKING CHANGE` | major |
| Any `feat:` | minor |
| Only fixes/chores | patch |

---

## Output Format

```markdown
## [X.Y.Z] — YYYY-MM-DD

### ⚠️ Breaking Changes
- Description (sha)

### Added
- Add I2C retry logic (a3f92bc)

### Fixed
- Fix null dereference on timeout (b2e41fc)

### Changed
- Refactor auth module structure (c1d52ae)

### Security
- Remove hardcoded credentials (d3f63bd)
```

Rules: strip type prefix, capitalize, include short SHA, omit empty sections.

---

## DRY_RUN

`true` (default): print to stdout only. `false`: write/prepend to CHANGELOG.md. Never commits.
