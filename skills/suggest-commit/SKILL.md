# Commit and PR Quality Plan (For Humans and Models)

## Purpose
This plan standardizes how we write commits and PRs so future debugging is easier as the codebase and team grow.

Primary goals:
- Always capture the **why** behind a change, not only the **what**.
- Keep each PR focused on **one task/ticket** unless combined items are tightly related.

## Core Rules

### 1) Explain Why in Commits and PRs
Every commit and PR should make clear:
- what changed (briefly)
- why it changed (the driving reason, constraint, or decision)
- impact/risk (what behavior this protects or could affect)

Bad:
- `update x to y`

Good:
- `update x to y because API parameter name changed to keep request parsing compatible`

### 2) One Task/Ticket Per PR
Default rule:
- one PR = one ticket/task

Allowed exception:
- multiple changes only when they are directly coupled and required to deliver one outcome.

If unrelated work appears during implementation:
- create a follow-up ticket/PR instead of bundling.

## Commit Message Template
Use this structure:

```
<type>: <short title>

Why:
- <business/technical reason for change>
- <constraint, regression, or dependency that motivated it>

What:
- <key change 1>
- <key change 2>

Impact:
- <behavior now enabled/protected>
- <known risk or migration note, if any>
```

### Commit Types (suggested)
- `feat` new behavior or UX
- `fix` bug fix or regression repair
- `refactor` internal change with no intended behavior change
- `docs` documentation-only updates
- `test` test-only updates
- `chore` maintenance/tooling

## PR Description Template
Use this structure in every PR:

```
## Why
- <why this work is needed now>
- <context when no linear ticket is attached>

## What Changed
- <change 1>
- <change 2>

## Scope
- Ticket/Task: <id or short description>
- Out of scope: <explicitly list what is not included>

## Risk & Rollback
- Risk level: Low / Medium / High
- Potential side effects: <areas to watch>
- Rollback plan: <how to safely revert or disable>

## Test Plan
- [ ] <test case 1>
- [ ] <test case 2>
- [ ] <regression path>
```

## Pre-Commit Checklist
- [ ] I included **why** (not just what changed).
- [ ] Commit title is specific and searchable.
- [ ] I avoided mixing unrelated edits.
- [ ] If API contract changed, I explained downstream implications.

## Pre-PR Checklist
- [ ] PR contains one task/ticket (or tightly-coupled exceptions only).
- [ ] PR description includes explicit **Why** section.
- [ ] Risks and rollback notes are documented.
- [ ] Test plan includes normal and edge/regression paths.

## Model Usage Guidance
When using AI models for commits/PR drafts:
- Ask for commit/PR text that includes **why, what, and impact**.
- Reject generic summaries that only list file edits.
- Require explicit mention of API contract changes and behavior effects.
- Split unrelated generated changes into separate PRs before review.

Prompt snippet for models:

```
Draft a commit/PR message using:
1) Why this change is needed
2) What changed
3) Impact/risk
Do not provide only file-level "what changed" bullets.
Also confirm this PR contains exactly one ticket/task or explain why exceptions are tightly coupled.
```

## Example

### Weak
`fix: updated generate payload`

### Strong
`fix: align generate payload with renamed API enhancePrompt parameter`

Why:
- Backend contract renamed parameter and old key caused silent fallback behavior.

What:
- Updated frontend request mapping and load-state handling for enhanced prompts.

Impact:
- Prevents incorrect prompt enhancement state after loading prior generations.
- Reduces repeated user toggling and avoids regression in edit/generate flow.
