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

- add login with PASETO access tokens, refresh sessions, and estate role lookup (2026-08-29)
- `AuthService.Login`: resolves a user by phone or email, verifies the bcrypt hash, issues an access token plus a refresh session recorded with the caller's user agent and IP, and returns the user's estates with the roles held in each. (2026-08-29)
- Token configuration: `TOKEN_SYMMETRIC_KEY`, `ACCESS_TOKEN_TTL` (default 15m) and `REFRESH_TOKEN_TTL` (default 168h), validated in `config.Load` so a wrong-sized key or an access TTL longer than the refresh TTL fails at startup rather than on the first login. (2026-08-29)
- `hash.HashToken`: SHA-256 digest for refresh tokens, since bcrypt is salted and the session lookup needs a deterministic hash. (2026-08-29)
- Auth repository methods `GetActiveSessionByTokenHash`, `RevokeSession` and `RevokeAllSessionsForUser`, backing refresh and logout. (2026-08-29)
- `ListUserEstateMemberships` query joining estates via `sqlc.embed`, grouped in the repository into one entry per estate carrying every role the user holds there — one round trip for the post-login estate picker. (2026-08-29)
- `SetEstateActive` query to enable or disable an estate without touching its memberships. (2026-08-29)
- `GetUserByPhone` on the user repository, so login accepts a phone number as the identifier. (2026-08-29)
- `web.RespondWithError`: maps domain errors to HTTP statuses in one place instead of a per-handler switch. (2026-08-29)
- Tests: PASETO maker (round trip, expiry, tampering, wrong key), bcrypt hasher, and the onboarding service and handler. (2026-08-29)
- add estate onboarding endpoint with domain repositories and auth primitives (2026-08-09)
- add web package with JSON response, request decode, and error helpers (2026-07-04)

### Changed

- added the readmme (2026-08-30)
- Session and account-claim-token integration tests in `sessions_test.go`: sessions (`TestCreateSession`, `TestGetSessionById`, `TestGetSessionByUserId`, `TestRevokeSession`, `TestRevokeAllSessionsForUser`), active-session lookups including not-found and expired negative cases, and claim tokens (`TestCreateAccountClaimToken`, `TestGetActiveClaimToken`). (2026-07-03)
- `GetSessionByUserId` (`:many`) query and regenerated sqlc output. (2026-07-03)
- `utils.RandomNumber` helper to spread session fixtures across users. (2026-07-03)
- `token.Payload` reduced to token id, user id, issued-at and expires-at. Roles and estate are resolved per request from the database, so a revoked membership stops working immediately instead of surviving until the token expires. (2026-08-29)
- `token.Maker.CreateToken` now returns the payload alongside the token string, so callers report the token's own expiry rather than recomputing it from a second clock reading. (2026-08-29)

### Fixed

- Estate and user repositories map `pgx.ErrNoRows` to `ErrEstateNotFound` and `ErrUserNotFound`, so a missing row surfaces as a 404 rather than a 500. (2026-08-29)

## [1.1.0] - 2026-06-28

### Added

- add membership queries and identity/session query tests (2026-06-28)
- Membership write queries: `CreateMembership` (link a user, estate, and role), `SuspendMembershipByID` (suspend one active membership by id), and `SuspendMembershipForUser` (suspend all active memberships for a user). (2026-06-28)
- Membership integration tests: `createRandomMembership` helper, `TestCreateMembership`, `TestListEstateMembershipForUser` (seeds distinct roles to respect the `UNIQUE (estate_id, user_id, role)` constraint), and `TestSuspendMembershipById`. (2026-06-28)
- Session integration tests in `sessions_test.go`: `createRandomSession` helper, `TestCreateSession`, and `TestGetSessionById`. (2026-06-28)

### Changed

- stop tracking .env (2026-06-28)
### Fixed

## [1.0.0]-(2026-06-18)

### Added

- DB-level pagination for estates: `ListEstates` now takes `page_size`/`page_offset`, and a new `CountEstates` query returns the total for page math. (2026-06-18)
- Integration tests for the identity and estate query layers under `internal/platform/db/dbtest`. (2026-06-18)
- `utils.RandomEstateAndSlug` test helper for generating unique estate fixtures. (2026-06-18)

### Changed

- ne (2026-06-18)
- first commit -m (2026-06-18)
- `config.Load` now walks up the directory tree to locate `.env`, so it works when `go test` runs from a sub-package directory. (2026-06-18)

### Fixed

- Estate query tests no longer fail with "no rows in result set" — fixtures now share the test's transaction instead of opening a separate one. (2026-06-18)
