---
name: review-pr
description: Deep senior-developer PR review for the aimkt-api project. Covers architecture, code quality, patterns, and implementation details specific to this Go / chi / sqlc / asynq backend codebase. Use when reviewing PRs or asked to review code changes.
argument-hint: "[pr-number]"
allowed-tools: Bash(gh *) Bash(git *) Read Grep Glob Bash(gh pr:*)
---

# PR Review — AI Marketer (Backend)

Review the pull request for this project. If a PR number is provided as `$ARGUMENTS`, fetch that PR. Otherwise fetch the current branch's open PR.

For repo-wide context, conventions, and the planned project layout, read [../../../CLAUDE.md](../../../CLAUDE.md) and [../checklist.md](../checklist.md). The product/backend contract lives in [../../../PRD_AI_Marketer_Backend.md](../../../PRD_AI_Marketer_Backend.md).

---

## Phase 1 — Fetch PR Context

### Get PR metadata

```!
if [ -n "$ARGUMENTS" ]; then
  gh pr view "$ARGUMENTS" --json number,title,body,headRefName,baseRefName,author,additions,deletions,changedFiles,files
else
  gh pr view --json number,title,body,headRefName,baseRefName,author,additions,deletions,changedFiles,files 2>/dev/null || echo "No open PR for current branch"
fi
```

### Get the diff

```!
if [ -n "$ARGUMENTS" ]; then
  gh pr diff "$ARGUMENTS"
else
  gh pr diff 2>/dev/null || git diff main...HEAD
fi
```

If the repo's default base branch is something other than `main`, substitute it.

### Read changed files in full

For every file touched in the diff, use the Read tool to read the **full file** — not just the changed lines. The diff alone misses context: what a function does elsewhere in the file, what neighbouring code looks like, whether a new helper duplicates something that already exists, and whether the change is consistent with the rest of the file.

For SQL changes in `queries/*.sql`, also read the corresponding sqlc-generated file in `internal/db/` to confirm the generated signature matches what handlers consume. For migrations, read both `up` and `down`.

---

## Phase 2 — Understand the PR

Summarise in 2–4 sentences:
- What does this PR do?
- What is the user-facing or operational impact (new endpoint, new job, schema change, provider swap)?
- What areas of the codebase does it touch (`internal/handlers`, `internal/db`, `internal/meta`, migrations, etc.)?

---

## Phase 3 — Architecture & Project Conventions

Check the changed files against the project's structural conventions. Read [../checklist.md](../checklist.md) for the full list. Flag any violation with file path and line number.

Key areas to check:
- Is new code in the right package — handlers in `internal/handlers`, DB code generated into `internal/db` from `queries/*.sql`, Meta calls only in `internal/meta`, AI calls only in `internal/ai`, S3 only in `internal/storage`, email only in `internal/email`, crypto helpers only in `internal/crypto`?
- Are external vendors hidden behind the right interface (`Publisher`, `CommentReader`, `MessageReader`, `Insights`, `LLMProvider`, `ImageGenProvider`, `MediaStorage`, `JobQueue`)? Business logic must not import a vendor SDK directly.
- Are new HTTP handlers thin — validate, call a service, write a response — with logic delegated to packages, not stuffed inline?
- Does every new protected route mount under the auth middleware (`RequireAuth`, `RequireRole("admin")` for admin routes, `RequireOrgRole(...)` later for team management)?
- Are admin endpoints on a separately-routed admin sub-router with the `admin` claim required?
- Are new SQL queries written in `queries/*.sql` and accessed through sqlc-generated code, not ad-hoc `db.Query(...)` strings?
- Do new tables/columns enforce tenancy at the schema level (`org_id` FK on every business table)?
- Do new asynq jobs go through the `JobQueue` interface, not asynq directly?
- Do new env vars land in [internal/config/config.go](../../../internal/config/config.go) with `envconfig` tags and `.env.example` updated?

---

## Phase 4 — Code Quality Deep Dive

Go beyond the checklist. Read the actual implementation and evaluate whether the approach is correct.

### Logic & Correctness
- Does the implementation actually do what the PR description claims?
- Are there off-by-one errors, incorrect comparisons, or wrong assumptions about data shapes?
- Are conditional branches exhaustive — are there unhandled cases that could cause silent failures?
- Is error handling meaningful — does it distinguish recoverable from fatal errors, or treat everything the same way?
- Are time/timezone calculations explicit (UTC at the boundary, scheduled-time accuracy)? Anything implicit local time is suspicious given the 60s p99 publish target.
- Are concurrent code paths goroutine-safe — shared maps protected, channels closed by the sender, no goroutines leaked beyond the request lifetime?

### Implementation Quality
- Is the approach the right one for the problem, or a workaround that works coincidentally?
- Is there unnecessary complexity — loops that could be a single query, repeated DB round-trips that could be one statement, multiple passes that could be one?
- Are there magic numbers or strings that should be named constants or come from config (TTLs, retry counts, token lifetimes, page sizes)?
- Does the code handle edge cases (empty slices, `nil` pointers, zero values, max input sizes, `sql.ErrNoRows`)?
- Is there dead code — variables assigned but never used, unreachable branches, unused returns?

### Abstraction & Duplication
- Does the change introduce logic that already exists somewhere in the codebase? Use Grep to check before concluding it's new.
- Is the abstraction level consistent — does a function mix HTTP-layer concerns with DB-layer concerns?
- Are new helpers genuinely reusable, or premature abstractions for a single call site?
- Does the naming describe the capability? Interfaces are noun-phrases (`Publisher`, not `MetaService`); helpers are verb-phrases.

### Go Idioms
- Does every function that does I/O take `ctx context.Context` as its first parameter, and is the request's context propagated all the way down?
- Are errors wrapped with `%w` (not `%v`/`%s`) so callers can `errors.Is` / `errors.As`?
- Are sentinel errors defined as exported variables when the caller needs to switch on them?
- Are exported names properly documented with godoc on `internal/<pkg>` types and functions?
- Are `defer rows.Close()` / `defer tx.Rollback()` / `defer resp.Body.Close()` present where needed?
- Is the zero value of a new struct usable, or does it require an explicit constructor?
- Are slices and maps preallocated with capacity when the size is known?
- Are interfaces accepted, structs returned (where it fits)?
- Are receiver names consistent (one or two letters, same name across all methods of a type)?

### Tenancy & Authorization (highest priority)
- Does every sqlc query for an org-owned resource take `org_id` as a parameter and use it in the `WHERE` clause? A missing `org_id` is a critical leak.
- Does every mutating handler resolve `org_id` from `OrgFromCtx(r)` (set by `RequireAuth`) and never trust it from the request body / URL?
- Does every new protected endpoint have a cross-org leak test using a two-org fixture?
- Are admin endpoints separately routed and require an admin claim — never co-located with regular org endpoints?
- Do new tables include `org_id` and the appropriate index?

### HTTP Layer (chi)
- Are new routes mounted with the right middleware order (rate-limit → auth → tenancy)?
- Are response status codes correct: `402` for feature-gate failures with an upgrade hint payload; `429` with `Retry-After`; `403` with `{redirect_to: "/onboarding/<step>"}` for incomplete onboarding; `409` for idempotent-conflict; `4xx` for client errors; `5xx` only for server faults?
- Are request bodies validated at the boundary (length, format, enum, count) before being passed to a service?
- Are responses encoded via `json.NewEncoder(w).Encode(...)` with `Content-Type` set?
- Does the handler ever stream a binary upload through itself? Media uploads must use presigned S3 PUT URLs via `MediaStorage.PresignUpload`.
- Does any handler block on an LLM/Meta call without enforcing the per-call timeout budget (LLM ≤8s p95, API non-AI ≤500ms p95)?

### Database (Postgres + sqlc + pgx)
- Are queries written so they hit an existing index, given the <50ms p95 query budget? New query patterns may need a new index.
- Are transactions used when multiple writes must be atomic (e.g. signup creating user + org + membership)?
- Is `ON CONFLICT` / upsert behaviour correct and idempotent where the operation can be retried?
- Are `nil` and `sql.ErrNoRows` translated to a meaningful API response (404 for missing, not 500)?
- Are migrations reversible (a sensible `down` step), and is any column-add on a hot table backfilled in a separate migration?
- Are large enums or seed data versioned via seed migrations rather than hard-coded in Go?

### External Integrations (Provider Portability)
- Every Meta Graph call goes through `internal/meta` (`Publisher` / `CommentReader` / `MessageReader` / `Insights`). No `graph.facebook.com` URLs in handlers.
- Every LLM call goes through `LLMProvider`. No `anthropic.` / `openai.` SDK calls in business logic.
- Every image-gen call goes through `ImageGenProvider`. No `recraft.` / `dalle.` calls outside `internal/ai`.
- Every storage call goes through `MediaStorage`. No `s3.` calls outside `internal/storage`.
- Every email goes through the email package's Resend wrapper.
- Stripe and Paystack each have their own internal client. Webhook handlers verify signatures and are idempotent on the provider event id.
- Provider selection comes from env config (`LLM_PROVIDER`, etc.), not hard-coded.

### Auth & Crypto
- Passwords hashed with argon2id (memory=64MB, iterations=3, parallelism=2) — never bcrypt or sha-anything.
- PASETO v4 (Ed25519) only — no JWT.
- Access tokens 15 min; refresh tokens 30d (90d with "remember me"), stored hashed in DB for revocation.
- Refresh-token cookies are HttpOnly + Secure + SameSite=Lax.
- Reset / verification tokens are random ≥32 bytes, hashed at rest, single-use, with a TTL (30 min for reset, 24h for email verify).
- Password reset revokes all existing refresh tokens for the user.
- All OAuth tokens (Meta, Google) encrypted at rest with AES-256-GCM via the `internal/crypto` helpers using a key from Fly secrets. Tokens are never logged, never returned to the client.

### Jobs (asynq)
- Are jobs routed through the `JobQueue` interface, not asynq directly?
- Do publish jobs use the 3-retry exponential-backoff policy? On final failure, is the draft marked `failed` and the user notified?
- Are job payloads small (IDs, not full entities) so Redis stays lean?
- Are scheduled cron jobs registered once at worker startup, with idempotent handlers (since asynq may redeliver)?
- Is the `ctx` honoured inside the handler so a graceful shutdown can cancel cleanly?

### Observability
- Is every log line `slog` with structured key/value pairs? No `fmt.Println`, `log.Printf`, or `println` in production paths.
- Do logs include `request_id`, `user_id`, `org_id` where applicable? Never `email`, `name`, or any other PII.
- Are uncaught panics, failed jobs, and explicit error captures reported to Sentry with org context (when wired)?
- Does every state-changing action write an `audit_log` row with `{org_id, actor_user_id, actor_type, action, entity_type, entity_id, metadata, created_at}`?
- Does the action emit metrics for the budgets it must hit (publish-timing accuracy, LLM latency)?

### Security
- Are secrets only ever read from Fly secrets / env, never committed?
- Are CORS, security headers (CSP default-deny, HSTS, X-Frame-Options, X-Content-Type-Options), and rate limits applied to new routes?
- Is rate limiting tighter on `/auth/*` and `/ai/*`?
- Are webhooks signature-verified and idempotent on the provider event id?
- Are user-supplied URLs / paths / SQL identifiers validated to prevent SSRF / path traversal / injection?
- Is the `404 vs unauthorized` distinction handled to avoid resource enumeration on org-owned data?

### Performance
- Is the new code within the budgets ([NFR-BE-2](../../../PRD_AI_Marketer_Backend.md)): API p95 <500ms, DB query p95 <50ms, LLM p95 <8s, publish accuracy p99 within 60s?
- Are N+1 query patterns avoided?
- Are slices/maps preallocated where size is known?
- Is HTTP / DB connection pooling configured rather than recreated per call?
- Any `time.Sleep` in production code? Suspicious unless explicitly justified.
- Any `fmt.Println` or debug code left in?

---

## Phase 5 — Issue Report

For each issue found, write a clear item:

**[SEVERITY] file path:line — Issue title**
> What the problem is, why it matters, and the concrete fix you'd recommend. Include a code snippet where it helps clarity.

Severity levels:
- **[BLOCKER]** — Must fix before merge. Bug, security risk, data integrity issue, cross-tenant leak, breaks existing behaviour.
- **[IMPORTANT]** — Should fix. Wrong pattern, missing edge case, risky approach, or tech debt that will compound.
- **[SUGGESTION]** — Nice to have. Cleaner code, better performance, more consistent with conventions.
- **[QUESTION]** — Needs clarification from the author before you can judge.

Only flag issues that are actually present in the code. Do not invent hypothetical problems. Be direct.

---

## Phase 6 — Summary

**What's done well** — Call out anything genuinely good: a well-structured approach, a well-named abstraction, correct handling of a tricky edge case, a careful migration. This isn't padding — it signals what to keep doing.

**Verdict** — one of:
- ✅ **Approve**
- ✅ **Approve with minor follow-ups** (list them)
- 🔁 **Request Changes** (list blockers)
- ❓ **Needs Discussion** (list open questions)

One short paragraph summarising your overall impression: what's good about this PR and what's the most important thing to address.
