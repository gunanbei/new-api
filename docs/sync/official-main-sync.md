---
name: official-main-sync
description: Audit functional changes from an official main ref and port only missing behavior into the current branch. Maintain this document's sync ledger so later runs use the last verified upstream tip as their incremental baseline.
---

# Official Main Sync

Use this skill when syncing the official main branch, or an explicitly named official upstream ref, into the current development branch. Port confirmed missing behavior, not commits wholesale. Preserve the current branch design where implementations conflict.

## Inputs

- <upstream>: official ref to inspect, normally main or origin/main.
- <target>: current branch; do not switch it implicitly.
- Optional scope: paths, components, or feature areas requested by the user.

## Scope

Include user-visible features, API behavior, bug fixes, protocol/provider support, data migrations, and billing or security behavior. Exclude formatting-only diffs, comments, lockfiles, behavior-neutral refactors, and unrelated configuration changes.

Do not call a change already ported merely because its commit is absent. Patch identity is a clue; equivalent observable behavior is the decision criterion.

## Procedure

### 1. Establish the starting state

    git status --short
    git branch --show-current
    git rev-parse HEAD
    git rev-parse <upstream>

Keep unrelated local changes intact. Record the target branch, its starting HEAD, and the immutable upstream tip before editing. Fetch only when needed and permitted; never silently move the target branch.

### 2. Select the review range

First inspect the latest **Completed** entry in [Sync History](#sync-history) whose source ref and target branch match this run.

    git merge-base --is-ancestor <recorded-source-end> <upstream-tip>

If it succeeds, review <recorded-source-end>..<upstream-tip>. This is the normal incremental path.

If there is no matching completed entry, the recorded upstream tip is no longer an ancestor (force-push/rewrite), or the last run is incomplete, establish a baseline from the common ancestor instead:

    base=$(git merge-base HEAD <upstream>)
    git log --reverse --format='%H%x09%s' "$base..<upstream>"

Record why the fallback was used. Do not use a previous target commit as an upstream baseline unless its ledger entry names the corresponding upstream end SHA.

### 3. Build a functional inventory

Inspect the range by logical behavior, not one commit at a time:

    git log --reverse --format='%H%x09%s' <range>
    git diff --name-status <range>
    git diff --stat <range>
    git cherry -v HEAD <upstream>

For each candidate, read the upstream change and the target call path. git cherry can identify likely equivalent patches but cannot prove behavioral equivalence.

Use this triage table in the run record:

| Candidate | User-visible behavior | Upstream files/contracts | Target equivalent? | Decision |
|---|---|---|---|---|
| ... | ... | DTO/route/model/config/callers | Yes/No, evidence | Port / Adapt / Skip / Defer |

For every functional candidate, confirm all of the following before editing:

1. Observable behavior: UI, API response, provider protocol, data persistence, or operational outcome.
2. Files and call paths affected, including callers and error paths.
3. DTO, route, interface, schema/migration, configuration, and compatibility implications.
4. Dependencies on adjacent upstream changes and whether the target already provides an equivalent result.
5. Security, billing, and cross-database implications where applicable.

Classify each candidate as:

| Status | Meaning |
|---|---|
| Already equivalent | Target already delivers the same observable behavior; document the evidence. |
| Missing, portable | Required behavior is absent and can be added without changing target design. |
| Adapt required | Behavior is missing but upstream structure conflicts with target design; preserve target architecture and make the smallest compatible change. |
| Excluded | Nonfunctional change within the stated exclusions. |
| Deferred | Functional but unsafe to isolate; name missing prerequisites, owner/scope, and reason. |

### 4. Port only confirmed behavior

Implement the smallest coherent patch. Do not cherry-pick broad refactors merely to obtain a feature. When upstream and target conflict, keep the current branch established architecture and adapt the feature at its integration boundary.

Before editing, read all target files and direct callers. Add focused deterministic tests for newly introduced behavior or a regression boundary; do not add coverage-only tests.

Apply project rules relevant to the area:

- Go JSON operations use common JSON wrappers; database code remains SQLite, MySQL, and PostgreSQL compatible.
- Relay request DTOs preserve explicitly supplied zero values with optional pointer fields.
- Billing changes validate multipliers and preserve quota-saturation, pre-consume, and settlement invariants.
- Frontend strings use useTranslation() and supported locale files; use Bun in web/default.

### 5. Run proportional regression checks

Run the checks that cover every touched area, then broaden when the change crosses shared contracts. Record exact commands and outcomes; a blocked command is not a pass.

| Touched area | Required checks |
|---|---|
| Any change | git diff --check; review the final diff and git status --short. |
| Go files | gofmt -w <changed-go-files>; go test <affected packages>. Run go test ./... or a meaningful package group when dependencies/environment allow. |
| Frontend (web/default) | Run applicable bun run lint, bun run format:check, and bun run build. Run bun run i18n:sync when user-visible strings changed. |
| Model/migration/database | Exercise the changed model path and reason explicitly about SQLite, MySQL, and PostgreSQL behavior; add a regression test where practical. |
| Relay/provider | Test request conversion, dispatch/error behavior, streaming where applicable, and explicit zero-value semantics. |
| Billing/quota | Test the request boundary and charge path: bounds, no negative charge, saturation audit, pre-consume, and settlement/refund behavior. |

For infrastructure-only failures, such as missing modules or unavailable network, keep dependency files unchanged, run every feasible focused check, and write the exact blocker in the record. A sync is not fully verified until its required checks pass or the exception is explicitly accepted and documented.

### 6. Complete the record and advance the baseline

Update [Sync History](#sync-history) in the same change. A **Completed** entry must contain:

- source ref, source start (exclusive), and source end (inclusive) SHAs;
- target branch, target HEAD before the work, and resulting commit SHA (or uncommitted);
- every ported feature and every skipped/deferred functional candidate with rationale;
- validation commands and results, including blockers;
- enough detail to decide the next incremental range without rediscovering a merge base.

Only mark the entry Completed after the reviewed range has no unclassified functional candidates. Deferred candidates are allowed only when explicitly named with a reason. The source-end SHA becomes the next run incremental baseline even when target commits are rebased; if upstream history was rewritten, fall back as described above and create a new baseline.

## Sync Record Template

Copy this block for each run, then replace every placeholder before marking it completed.

~~~markdown
### YYYY-MM-DD - <id> - Completed | In progress | Blocked

- Source: <upstream>; range: <source-start>..<source-end>
- Target: <branch>; start: <target-head-before>; result: <target-head-after|uncommitted>
- Baseline: previous ledger entry | merge-base fallback (<reason>)
- Ported:
  - <behavior> - <files> - <test/verification>
- Already equivalent / excluded:
  - <behavior or change> - <evidence or exclusion reason>
- Deferred:
  - <behavior> - <prerequisites and reason>
- Validation:
  - command - passed | failed | blocked: reason
- Known blockers: <none | details>
~~~

## Sync History

Keep newest entries first. The table is an index; each detailed entry below it is the authoritative evidence and range baseline.

| Date | ID | Status | Source range | Target branch | Result | Notes |
|---|---|---|---|---|---|---|
| 2026-07-27 | 001 | Completed with documented blocker | 1086038f5f893a4558366f4d314cabc4ef5c8a23..60a1acb703a64186bf6eeef441e2fac947b75f26 | codex/develop-tmp | uncommitted | Functional review completed; broader controller/service tests blocked by an existing missing module checksum. |

### 2026-07-27 - 001 - Completed

- Source: main; range: 1086038f5f893a4558366f4d314cabc4ef5c8a23..60a1acb703a64186bf6eeef441e2fac947b75f26
- Target: codex/develop-tmp; start: 26d7e052bee26adb02d49064829273ab16f4ac86; result: uncommitted
- Baseline: merge-base fallback (no prior ledger entry)
- Ported:
  - Auto-group model discovery with ordered, deduplicated group models - controller/model.go, controller/user.go, service/group.go - controller/model_list_test.go
  - OpenAI Realtime GA header handling and model names - relay/channel/openai/adaptor.go, relay/channel/openai/constant.go - relay/channel/openai/adaptor_realtime_test.go
  - Gemini image GA model names and default imagine support - relay/channel/gemini/constant.go, setting/model_setting/gemini.go
  - Tencent TokenHub dispatch support - relay/relay_adaptor.go, relay/channel/tencent/dispatch.go - relay/channel/tencent/dispatch_test.go
  - Upstream error-body diagnostics when a parsed error message is empty - service/error.go
- Already equivalent / excluded:
  - Task refund CAS protection - target already had equivalent protection.
  - Formatting, comments, lockfiles, behavior-neutral refactors, and unrelated configuration - excluded by scope.
- Deferred:
  - Auth/session replacement, relaykit extraction, New API/Sub2API, alpha-search, configurable tool billing, and advanced Custom upstream-model discovery - coupled changes with conflicting or incomplete target integration; require dedicated integration work.
- Validation:
  - git diff --check - passed
  - GOCACHE=/private/tmp/new-api-go-cache go test ./relay/channel/openai ./relay/channel/tencent ./relay/channel/gemini - passed
  - go test for controller/service coverage - blocked: existing missing go.sum entry for gorm.io/driver/sqlite imported by controller/creative_execution_test.go; network/DNS to proxy.golang.org unavailable, and dependency files were not changed.
- Known blockers: controller and service package regression suite remains pending until the existing module-checksum/network issue is resolved.
