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
| 2026-08-03 | 003 | Completed | f3ab2cff36b3962815be9114e300d26927cc42b3..0ab02020603d22e5613bc4cf46bfab06f8567769 | develop_tmp | uncommitted | Ported the isolated provider, billing, channel transport, and token Auto-group behavior; three coupled relay/session changes remain deferred. |
| 2026-07-27 | 002 | Completed | 60a1acb703a64186bf6eeef441e2fac947b75f26..f3ab2cff36b3962815be9114e300d26927cc42b3 | codex/develop-tmp | uncommitted | Ported the GitCode release-sync workflow; remaining changes were behavior-neutral refactors. |
| 2026-07-27 | 001 | Completed with documented blocker | 1086038f5f893a4558366f4d314cabc4ef5c8a23..60a1acb703a64186bf6eeef441e2fac947b75f26 | codex/develop-tmp | uncommitted | Functional review completed; broader controller/service tests blocked by an existing missing module checksum. |

### 2026-08-03 - 003 - Completed

- Source: origin/main; range: f3ab2cff36b3962815be9114e300d26927cc42b3..0ab02020603d22e5613bc4cf46bfab06f8567769
- Target: develop_tmp; start: dd26f1d358d520a8f667fe7270945570682029d8; result: uncommitted
- Baseline: previous ledger entry; its source end is an ancestor of the reviewed origin/main tip.
- Triage:

  | Candidate | User-visible behavior | Upstream files/contracts | Target equivalent? | Decision |
  |---|---|---|---|---|
  | Gemini OpenAI-chat stream terminal conversion | Normalizes terminal state for converted Gemini streams | relay stream converter and shared relay state | No, but coupled to conflicting relay stream architecture | Defer |
  | iPad login session detection | Correctly recognizes iPad sessions during login | frontend auth/session detection | No, but requires broader session-browser compatibility review | Defer |
  | Per-channel HTTP transport controls | Select HTTP negotiation mode and HTTP/2 connection shards per channel | channel setting JSON, transport client cache, API/AWS/Coze/Vertex callers, channel form | No | Port / Adapt |
  | OIDC display name | Allows an administrator to label the OIDC login option | option, status API, auth settings form | No | Port |
  | Provider and relay fixes | zstd request handling, Qwen thinking budget passthrough, DeepSeek Responses conversion, stream-status log visibility, OAuth opener binding, Bedrock cancellation | middleware, relay DTO/adaptors, logs, OAuth | No | Port |
  | Retry billing fixes | Uses final group and bounded retry accounting during tiered settlement | billing usage and tiered settlement | No | Port |
  | Token-specific Auto group order | Persists a per-token ordered Auto snapshot with an administrator-set cap | token model/controller/routes, group selection, settings, key and admin forms | No | Port / Adapt |
  | GORM SQL logging redaction | Avoids exposing values in slow-query and error logs | model logger | No | Port |
  | New API multipart image edits | Preserves multipart image-edit payloads | relay New API adaptor and request flow | No, coupled to a broader relay contract change | Defer |
  | CI, RelayKit README, .gitattributes, mutex/TCP test refactors, navigation text-size change | No product behavior in this sync scope | workflow, documentation, configuration, tests, styles | N/A | Excluded |
- Ported:
  - OIDC display-name option, status propagation, and translated admin UI - `setting/system_setting/oidc.go`, `controller/misc.go`, `web/default/src/features/system-settings/auth/oauth-section.tsx`.
  - zstd request decompression, Qwen `thinking_budget` passthrough, stream-status log visibility, OAuth opener/bind handling, and DeepSeek Responses conversion - middleware, relay, OAuth, and log UI paths.
  - Tiered retry billing settlement, client-disconnect cancellation for AWS Bedrock, and parameter-redacted GORM slow-query/error logging - `service/`, `relay/channel/aws/`, and `model/gorm_logger.go`.
  - Per-channel HTTP transport policy, sharded transports, and channel editor controls for automatic/HTTP/1.1 mode and HTTP/2 shard count - `dto/channel_settings.go`, `service/http_*`, relay callers, and channel form/UI files.
  - Token-specific Auto-group persistence and selection, user-scoped Auto-group API, maximum snapshot setting, and key/admin UI support - token model/controller/service/settings plus `web/default/src/features/keys/` and system settings forms.
- Already equivalent / excluded:
  - CI, RelayKit documentation, `.gitattributes`, mutex/TCP test refactors, and navigation typography were excluded as nonfunctional or outside the requested sync scope.
- Deferred:
  - Gemini OpenAI-chat terminal stream conversion - conflicts with the target stream conversion design and needs dedicated relay integration coverage.
  - iPad login-session detection - requires a dedicated compatibility review across existing browser/session detection paths.
  - New API multipart image edits - requires isolating the broader multipart relay contract before porting safely.
- Validation:
  - `gofmt -w` on touched Go files - passed
  - `cd web/default && bun run i18n:sync` - passed
  - `git diff --check` - passed
  - final diff and `git status --short` - reviewed
  - Go tests, frontend typecheck/lint/build, and manual browser tests - not run at user request.
- Known blockers: none; deferred items are intentionally outside this isolated sync.

### 2026-07-27 - 002 - Completed

- Source: origin/main; range: 60a1acb703a64186bf6eeef441e2fac947b75f26..f3ab2cff36b3962815be9114e300d26927cc42b3
- Target: codex/develop-tmp; start: 767ef3875d8581c48bbafbccf9fa8ea59a709d49; result: uncommitted
- Baseline: previous ledger entry; its source end is an ancestor of the fetched origin/main tip.
- Triage:

  | Candidate | User-visible behavior | Upstream files/contracts | Target equivalent? | Decision |
  |---|---|---|---|---|
  | GitCode release synchronization | Tags or manual dispatch publish missing GitHub release assets to GitCode without duplicating existing assets | `.github/workflows/sync-release-to-gitcode.yml`; requires `GITCODE_REPOSITORY` variable and `GITCODE_TOKEN` secret | No; target has no GitCode sync workflow | Port |
  | Types extraction and trusted-proxy relocation | None | Go import/package paths only | Yes; runtime behavior is unchanged | Excluded |
- Ported:
  - GitCode release synchronization, including upstream release completion waits, asset normalization/deduplication, bootstrap release creation, and matrix asset publishing - `.github/workflows/sync-release-to-gitcode.yml` - final workflow matches `origin/main`.
- Already equivalent / excluded:
  - Types extraction and trusted-proxy relocation - behavior-neutral refactors; no port required.
- Deferred:
  - None.
- Validation:
  - `git diff --check` - passed
  - `git diff --no-index <(git show origin/main:.github/workflows/sync-release-to-gitcode.yml) .github/workflows/sync-release-to-gitcode.yml` - passed
  - `ruby -e "require 'yaml'; YAML.load_file('.github/workflows/sync-release-to-gitcode.yml')"` - passed
  - final diff and `git status --short` - reviewed
- Known blockers: none.

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
