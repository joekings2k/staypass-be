# Changelog

All notable changes to the Staypass backend are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

<!--
  New entries are added automatically under [Unreleased] by the
  scripts/git-hooks/prepare-commit-msg hook on every commit.
  Run `make install-hooks` once after cloning to enable it.
  Commit subjects are sorted by their conventional-commit prefix:
    feat: -> Added   fix: -> Fixed   anything else -> Changed
-->

## [Unreleased]

### Added

- DB-level pagination for estates: `ListEstates` now takes `page_size`/`page_offset`, and a new `CountEstates` query returns the total for page math. (2026-06-18)
- Integration tests for the identity and estate query layers under `internal/platform/db/dbtest`. (2026-06-18)
- `utils.RandomEstateAndSlug` test helper for generating unique estate fixtures. (2026-06-18)

### Changed

- `config.Load` now walks up the directory tree to locate `.env`, so it works when `go test` runs from a sub-package directory. (2026-06-18)

### Fixed

- Estate query tests no longer fail with "no rows in result set" — fixtures now share the test's transaction instead of opening a separate one. (2026-06-18)
