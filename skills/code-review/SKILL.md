---
name: code-review
description: Quick code review before creating a PR. Catches common issues in your working branch changes against main — wrong patterns, missing error handling, obvious bugs, convention violations. Use before opening a PR so the full review-pr skill has less to flag.
allowed-tools: Bash(git *) Read Grep Glob
---

# Code Review — AI Marketer (Backend)

Run a quick review of your uncommitted or branch changes before creating a PR. This catches the most common issues so the deeper PR review has less to flag. For the full list of project conventions, see [../checklist.md](../checklist.md) and [../../../CLAUDE.md](../../../CLAUDE.md).

## Step 1 — Get the diff

```!
git diff main...HEAD --stat
```

```!
git diff main...HEAD
```

If the repo's default branch is something other than `main` (e.g. `dev`), substitute it.

## Step 2 — Quick scan

For each changed file, check the following. Be fast and direct — skip anything that looks fine and only flag actual problems.

### Must-fix (would block a PR)
- **Cross-tenant leak** — sqlc query for an org-owned resource missing `org_id` in `WHERE`, or handler reading/writing org data without resolving `org_id` from the request context (`OrgFromCtx`)
- **Missing auth** — protected handler not behind `RequireAuth` (or `RequireRole` for admin routes); admin endpoint exposed under a non-admin route
- **Missing authorization helper** — mutating handler writes org data without calling the central authorization helper to check role × action
- **Token / secret leak** — Meta/Google OAuth token, PASETO key, password hash, or any secret logged, returned in a response, or stored unencrypted in the DB
- **Plaintext OAuth tokens at rest** — Meta token persisted without going through the AES-256-GCM encrypt helper
- **Direct vendor SDK in business logic** — handler/service calling Meta Graph, Anthropic/OpenAI, S3, Resend, Stripe, or Paystack directly instead of through the corresponding interface (`Publisher`, `LLMProvider`, `MediaStorage`, etc.)
- **Webhook without signature verification or idempotency** — Stripe / Paystack / Meta webhook handler missing signature check, or able to double-process the same event id
- **Binary upload streamed through the API** — handler accepting multipart file bytes instead of issuing a presigned S3 URL via `MediaStorage.PresignUpload`
- **Wrong logger** — `fmt.Println`, `log.Printf`, or `println` in production code paths instead of `slog`
- **PII in logs** — emails, names, phone numbers, message bodies, or token values passed as `slog` attributes
- **Missing context** — long-running call (DB query, HTTP, LLM, Meta API) not taking `ctx context.Context` as the first arg, or ignoring `r.Context()`
- **Swallowed error** — `_ = someCall()` on a non-best-effort call (DB write, external API, transaction commit) with no surfacing
- **Missing audit log** — state-changing action (publish, approve, disconnect, generate, subscription change, admin mutation) not writing an `audit_log` row

### Should-fix (will likely come up in review)
- **Wrong package placement** — handler logic in `internal/db`, SQL strings outside `queries/`, Meta API calls outside `internal/meta`, AI calls outside `internal/ai`, storage calls outside `internal/storage`
- **Fat handler** — business logic inline in the HTTP handler instead of delegated to a service/package; handler should validate, call, and respond
- **Raw SQL outside sqlc** — ad-hoc `db.Query(...)` strings instead of generated sqlc functions from `queries/`
- **Missing HTTP timeouts** — `http.Server` constructed without `ReadHeaderTimeout` / `ReadTimeout` / `WriteTimeout` / `IdleTimeout`; outbound `http.Client` using default (no timeout)
- **Missing graceful shutdown** — new long-lived process (server, worker) without `SIGINT`/`SIGTERM` handling and bounded `Shutdown(ctx)`
- **Wrong status codes** — feature-gate failure not returning `402` with upgrade hint; rate-limit hit without `429` + `Retry-After`; incomplete onboarding not returning `403` with `{redirect_to}` payload
- **No retry / no backoff on publish job** — asynq publish handler not respecting the 3-retry exponential-backoff policy, or final failure not marking the draft `failed` and notifying the user
- **Missing rate-limit bucket** — new `/auth/*` or `/ai/*` endpoint not covered by the stricter token bucket
- **Cross-org test missing** — new protected endpoint added without a two-org fixture test proving cross-tenant isolation
- **`interface{}` / `any`** without a legitimate reason
- **Ignored `Close()` / `Rollback()`** — `defer rows.Close()` or `defer tx.Rollback()` missing on DB resources
- **Goroutine without lifetime** — `go func()` started in a request handler with no `ctx` and no way to cancel
- **Missing input validation** — request body decoded straight into a DB call without bounds / format / enum checks at the boundary
- **Magic numbers** — hard-coded TTLs, token lifetimes, retry counts that should come from config or named constants
- **`fmt.Errorf` without `%w`** — wrapping an error but losing the chain via `%v` / `%s`

### Worth mentioning
- **Naming** — does a function/struct/interface name accurately describe what it does? Interfaces should be capability nouns (`Publisher`, not `MetaService`)
- **Duplication** — does this logic already exist elsewhere? (Grep to check before adding a helper)
- **SQL index hint** — new query pattern on a table without a matching index, given the <50ms p95 budget
- **Missing godoc** on exported types/functions in `internal/<pkg>`
- **`make fmt vet` not run** — formatting drift or `go vet` warnings the diff would surface
- **Migration safety** — irreversible migration, missing `down`, or a column add on a hot table without a backfill plan

## Step 3 — Output

List only the issues you actually found. For each one:

**file:line — What's wrong**
> One sentence on why it matters and how to fix it.

Group by severity: must-fix first, then should-fix, then worth-mentioning.

If everything looks clean, say so — don't invent issues. End with a one-line summary: "Ready to open PR" or "Fix N issues first".
