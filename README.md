# Staypass — Backend

Gate access for Nigerian residential estates, without the paper visitor book.

This is the Go API behind Staypass. It is **work in progress** — the identity and
tenancy foundation is built and tested, the access-pass core is next. This README
describes both what runs today and what it is being built toward, so the shape of
the thing is clear before it is finished.

---

## The problem

A typical gated estate runs access control on a paper logbook at the gate. A
visitor writes a name, the gateman waves them through, and nobody can answer
basic questions afterwards: who is in the estate right now, who let them in, is
this person still a tenant here. Residents call the gate to announce guests.
Landlords have no record of who occupies their unit. When a lease ends, nothing
automatically stops the former tenant from walking back in.

## What Staypass does

A resident generates a time-boxed pass on their phone and sends it to their
visitor over WhatsApp. The gateman scans it. The scanner says allow or deny, and
the scan is logged — who, when, which gate, what outcome. Residents themselves
carry a pass derived from their active lease, so when the lease ends the pass
stops working on its own. No new hardware: the gate runs a PWA on a cheap Android
phone, and it keeps validating visitor passes when the network drops.

### The v1 flows

1. **Estate onboarding** — an admin registers the estate, adds units, assigns
   owners, registers residents into units.
2. **Resident pass** — every active resident gets a gate pass. Server-validated,
   with a short-lived signed token as the offline fallback.
3. **Visitor pass** — a resident generates a time-boxed, HMAC-signed pass and
   shares it as a `wa.me` deep link. Validates fully offline.
4. **Gate scanning** — the gateman scans either pass type, gets allow/deny, and
   every scan is recorded.
5. **Rental pipeline** — an owner lists a vacant unit, a prospect applies, the
   owner approves, and the agreement provisions the tenant account, the lease,
   and the resident pass in one step. A lean pipeline, not a marketplace.
6. **Operational overview** — the admin sees who is currently inside, occupancy
   per unit, and the access log.

### Roles

A person is modelled separately from what they do. Roles live on the
`memberships` join table, never on the user — so one person can be a landlord
*and* a resident in the same estate, and the same account can hold different
roles in different estates.

| Role | Can do |
|------|--------|
| **Super admin** | Platform-level; seeded by hand, no console in v1 |
| **Estate admin** | Onboards units, owners and residents; sees the operational overview |
| **Gateman** | Scans at the gate, allow/deny, nothing else |
| **Landlord** | Owns one or more units, can list a vacant one |
| **Resident** | Holds a resident pass, generates visitor passes |
| **Visitor** | No account — just receives a pass link |

---

## Architecture

| | |
|---|---|
| Language | Go 1.26 |
| Router | `chi/v5` |
| Database | PostgreSQL 16 (`pgx/v5`) |
| Queries | [sqlc](https://sqlc.dev) — hand-written SQL, generated type-safe Go |
| Migrations | [goose](https://github.com/pressly/goose) |
| Tokens | PASETO v2 local (`o1egl/paseto`) |
| Passwords | bcrypt; refresh tokens hashed with SHA-256 |
| Config | `envconfig` + `godotenv`, validated at startup |
| Tests | `testify`, `go.uber.org/mock` |

### Design decisions worth knowing

**Multi-tenant from day one, single estate in v1.** Every table carries
`estate_id`. Only one estate goes live first, but going multi-estate later is a
config switch rather than a rewrite. Shared database, shared schema, discriminator
column.

**Tokens carry identity, not authority.** `token.Payload` holds only the token
id, user id, issued-at and expires-at. Roles and estate are resolved from the
database on each request. It costs a query, but it means revoking a membership
takes effect immediately instead of surviving until the access token expires.

**Repositories return domain types, not database rows.** Each domain package
(`user`, `estate`, `membership`, `auth`) defines its own structs and maps
`pgx`/sqlc types at the repository boundary. Services never import generated
database code, so the persistence layer can change without touching business
logic.

**Multi-step writes run in one transaction.** `database.TransactionRunner` hands
a scoped `*dbgen.Queries` to a callback, so onboarding creates the estate, the
user and the admin membership atomically — a failure anywhere rolls all three
back.

**Three independent off switches.** `users.deleted_at` kills an account,
`memberships.status` removes one person from one estate, and `estates.is_active`
is a reversible tenant-level kill switch that leaves membership rows intact so
reactivation restores everyone instantly.

**Phone-first identity.** `users.phone` is the unique required identifier; email
is optional. `password_hash` is nullable because accounts can be provisioned by
an admin and claimed later, via a one-time `account_claim_tokens` link sent over
WhatsApp.

### Layout

```
cmd/api/                  entrypoint: config, pool, DI, graceful shutdown
internal/
  auth/                   sessions, login, refresh   (service built, not routed)
  estate/                 estate repository
  membership/             membership repository, role types
  onboarding/             estate + admin registration  (live)
  user/                   user repository
  server/                 router, middleware, handler wiring
  platform/
    config/               env loading + startup validation
    database/             pgx pool, transaction runner
    db/queries/           hand-written SQL  →  db/gen/ (sqlc output)
    db/dbtest/            integration tests against a real Postgres
    hash/                 bcrypt hasher, SHA-256 token hasher
    token/                PASETO maker, payload
    web/                  decode, validate, respond, error mapping, logging
    utils/                slugs, random test data
migrations/               goose migrations
```

---

## Status

Honest snapshot — this is a project in progress, not a finished service.

### Working

- **Schema**: `users`, `estates`, `memberships`, `sessions`,
  `account_claim_tokens`, with enum role/status types, `updated_at` triggers and
  a `UNIQUE (estate_id, user_id, role)` constraint enforcing one row per role.
- **Estate onboarding**: `POST /api/v1/estate-onboarding` creates the estate, the
  admin user and the `estate_admin` membership in a single transaction.
- **Health check**: `GET /health`, including a database ping.
- **Auth service**: login by phone *or* email, bcrypt verification, PASETO access
  token, refresh session recorded with user agent and IP, and the user's estates
  with the roles held in each returned for the post-login estate picker.
- **Session repository**: create, look up by id / user / active token hash,
  revoke one, revoke all for a user, delete expired.
- **Config validation**: a wrong-sized symmetric key or an access TTL longer than
  the refresh TTL fails at startup rather than on the first login.
- **Tests**: PASETO maker (round trip, expiry, tampering, wrong key), bcrypt
  hasher, onboarding service and handler, plus integration tests for identity,
  estates and sessions against a real database.

### In progress

- Auth HTTP layer — the service exists but `auth.Handler` is not yet mounted on
  the router, so login is not reachable over HTTP yet.
- Auth middleware — no route is protected yet; there is no token-verification
  middleware and no request-scoped role resolution.
- Refresh and logout endpoints on top of the session repository methods.

### Not started

- `units`, `leases`, `passes` and `scan_logs` tables — the access-pass core of the
  spec.
- HMAC visitor-pass signing, the WhatsApp `wa.me` delivery link, and the offline
  validation path.
- The gate scanner endpoints and the scan log.
- The rental pipeline (listing → application → approval → auto-provisioned tenant).
- The admin operational overview.
- Rate limiting on auth endpoints.

---

## Running it

**Prerequisites:** Go 1.26+, Docker, [goose](https://github.com/pressly/goose)
and [sqlc](https://sqlc.dev) on your `PATH`.

```bash
# create a .env in this directory with the variables listed below
docker compose up -d      # Postgres 16 on :5432
make migrate-up
make run                  # API on :8080
```

Check it is alive:

```bash
curl localhost:8080/health
```

Create an estate and its first admin:

```bash
curl -X POST localhost:8080/api/v1/estate-onboarding \
  -H 'Content-Type: application/json' \
  -d '{
    "fullName":   "Ada Obi",
    "email":      "ada@example.com",
    "phone":      "+2348012345678",
    "password":   "supersecret",
    "estateName": "Rainbow Court",
    "address":    "12 Admiralty Way",
    "city":       "Lekki",
    "state":      "Lagos"
  }'
```

### Environment

| Variable | Notes |
|---|---|
| `DATABASE_URL` | required |
| `TEST_DATABASE_URL` | separate database for the integration tests |
| `TOKEN_SYMMETRIC_KEY` | **exactly 32 bytes** — `openssl rand -hex 16` |
| `ACCESS_TOKEN_TTL` | Go duration, default `15m` |
| `REFRESH_TOKEN_TTL` | Go duration, default `168h` (7 days); must exceed the access TTL |
| `CORS_ALLOWED_ORIGINS` | required, comma-separated |
| `APP_ENV` | `development` (text logs) or `production` (JSON logs) |
| `APP_PORT` | default `8080` |
| `POSTGRES_*` | user, password, db, host, port — consumed by `docker-compose.yml` |

### Make targets

```
make run             # start the API
make test            # migrate the test database, then go test ./...
make sqlc            # regenerate db/gen from queries/ after editing SQL
make migrate-up      # apply migrations
make migrate-down    # roll back the last migration
make migrate-status  # show what has been applied
make migrate-new name=add_units
make install-hooks   # point git at scripts/git-hooks (auto-updates CHANGELOG.md)
```

Tests hit a real Postgres rather than mocking the driver — `make test` runs the
migrations against `TEST_DATABASE_URL` first.

---

## Roadmap

**Next**, to close out v1: mount the auth routes and middleware, then build the
access core — units and leases, the pass tables, HMAC signing, WhatsApp delivery,
and the gate scanner endpoints with the scan log.

**Deliberately deferred to v2:** dues and service-charge collection with Paystack,
dues-gated access, rent payment inside the rental flow, in-app owner/renter chat,
the multi-estate console and super-admin UI, per-estate branding, and any native
mobile app.

The full product definition lives in [`../estate-access-v1-spec.md`](../estate-access-v1-spec.md);
the change history is in [`CHANGELOG.md`](CHANGELOG.md).
