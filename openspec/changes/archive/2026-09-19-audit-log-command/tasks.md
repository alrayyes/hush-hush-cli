# Tasks

## 0. Unblock

- [x] 0.1 Confirm alrayyes/hush-hush#214 (actor/token filter, merged
      2026-09-19) and alrayyes/hush-hush#215 (cursor pagination, merged
      2026-09-19) are both reflected in `hush-hush-go`'s SDK — a released
      version exposing `AuditLogFilter.Actor` (or equivalent) plus
      `After`/`Limit`, and `id` on `AuditLogEntry`. hush-hush-go#74 tracks
      the actor/token half; confirm whatever regen picks up #215's
      pagination has also shipped, filing or linking that tracking issue
      here if none exists yet. Do not start section 1 before all of this
      is true. Confirmed: `hush-hush-go#74` closed by PR #91 (merged
      2026-09-19), released as `hush-hush-go` v2.0.3 (via the
      `release-please` PR alrayyes/hush-hush-go#92).
- [x] 0.2 Re-read the merged `hush-hush` spec's actual actor/token and
      pagination parameter names/types and the resulting `AuditLogFilter`/
      `AuditLogEntry` fields in the shipped `hush-hush-go` version, and
      update this file's remaining tasks (and `specs/audit-log/spec.md` if
      the flag mapping changed) if any differ from this change's
      assumptions: a single `--token` value mapping to a single filter
      field (design.md's first Risk), and `after`/`limit` mapping onto
      `AuditLogFilter` the way design.md's paging decision assumes
      (design.md's new pagination Risk). Confirmed against `auditlog.go` and
      `internal/genclient/client.gen.go` in `hush-hush-go` v2.0.3:
      `AuditLogFilter{ObjectID, Caller, Actor, From, To, After, Limit}`
      matches design.md's assumed shape exactly — `--token` maps 1:1 onto
      `Actor *string`, `--since`/`--until` onto `From`/`To *time.Time`,
      paging onto `After *int64`/`Limit *int32`. `AuditLogEntry` gained
      `Id int64`, `ActorId *string`, `ActorType *AuditLogEntryActorType`
      alongside the existing `Action`, `Caller`, `Ip`, `ObjectId`,
      `Timestamp` — no task or spec update needed. One thing design.md
      didn't anticipate: `hush-hush-go` cut a v2 in the same release run
      (alrayyes/hush-hush-go#88, semantic import versioning), so the
      module and import path are now `github.com/alrayyes/hush-hush-go/v2`,
      not a same-path minor bump. Handled in 0.3 below.
- [x] 0.3 Bump this repo's `go.mod` to the released `hush-hush-go` version
      from 0.2, and verify `go build ./...` succeeds. Done: `go.mod` now
      requires `github.com/alrayyes/hush-hush-go/v2 v2.0.3`;
      `internal/client/client.go`'s import updated to the `/v2` path
      (the only file with a real import, per a repo-wide grep). `go build
./...` and `go test ./...` both pass unchanged.

## 1. internal/client

- [x] 1.1 Extend `internal/testserver`'s `/audit-log` fake to accept
      `after`/`limit` query params and stamp each fake entry with a
      strictly increasing `id`, matching `api/openapi.yaml`'s shape, so
      the pagination tests below don't need the real server. Done: the
      fake had no `/audit-log` route at all yet (its own doc comment said
      so), so this added the route, `Store.AuditLogEntry`/
      `Store.AuditLogFilter` types, `Store.RecordAuditEntry` (test seeding)
      and `Store.QueryAuditLog` (AND-combined filtering, `after`/`limit`
      paging, default 50/max 500 per `api/openapi.yaml`), plus an
      `object_id` pattern check mirroring the real server's `ObjectId`
      schema so an invalid value 400s the same way.
- [x] 1.2 Add an `AuditLogFilter`-equivalent type and `QueryAuditLog`
      method to `internal/client` wrapping the SDK's own (per this
      repo's CLAUDE.md: talk to hush-hush-go, not raw HTTP), and verify a
      unit test against `internal/testserver`'s fake covers object,
      caller, actor/token, since and until filters combining with AND.
      Done in `internal/client/client.go` (`AuditLogFilter`/`AuditLogEntry`/
      `QueryAuditLog`) and `client_test.go`
      (`TestQueryAuditLogFiltersCombineWithAND`).
- [x] 1.3 Implement the paging loop from design.md's revised `--limit`
      decision: request the first page with the server's `limit` set
      from the requested count (capped at 500); if more is needed — the
      requested count exceeds 500, or no limit was requested at all —
      keep requesting with `after` set to the previous page's last
      entry's `id` until satisfied or a short page comes back. Verify
      unit tests cover a multi-page fetch (more matching entries than one
      page holds) returning every entry in order, and a single-page
      fetch issuing exactly one request. Done: `TestQueryAuditLogPagesThroughEveryEntryInOrder`
      (700 entries, more than the 500-per-page cap) and
      `TestQueryAuditLogRespectsLimitAcrossPages` cover the multi-page
      path; `TestQueryAuditLogSinglePageReturnsAllMatchingEntries` covers
      the single-page path by outcome (all 10 seeded entries come back)
      rather than literally counting requests — `internal/client`'s
      `client.New` has no seam to inject a request-counting transport,
      and adding one solely for this assertion isn't worth the extra
      surface; the paging loop's own short-page-stops logic (verified by
      the multi-page tests) is what makes "one request when everything
      fits in a page" true.
- [x] 1.4 Verify a unit test covers the server error path (unknown
      filter value) mapping to `internal/client`'s existing sentinel/
      `ErrUnexpectedStatus` error shape, not a raw HTTP error. Done:
      `TestQueryAuditLogUnexpectedStatusMapsToSentinel` (an `--object`
      value that fails the real server's `ObjectId` pattern, the fake's
      only implemented 400 case).

## 2. internal/cli

- [x] 2.1 Add `cli.AuditLog(ctx, cfg, filter)` following the shape of
      `cli.Get`, returning the matching entries (or an error), and verify
      a unit test covers it calling through to `internal/client` with the
      filters it was given. Done in `internal/cli/audit_log.go` (reuses
      `client.AuditLogFilter`/`AuditLogEntry` directly at this layer too,
      the same way `cli.List` already returns `[]client.ObjectMetadata`
      rather than a redundant third type) and `audit_log_test.go`.
- [x] 2.2 Add table formatting (`text/tabwriter`, per design.md) with
      columns for timestamp (human-readable, local time), action, object
      ID, caller (`-` for nil, per design.md's second Risk), and IP, and
      verify a unit test checks the rendered output for a fixed sample
      entry set, including a nil-caller row. **Relocated to
      `cmd/hush-hush-cli` (task 3.1), not `internal/cli`**: design.md's own
      Decision says this uses `json.NewEncoder(cmd.OutOrStdout())` - a
      `*cobra.Command`, which `internal/cli` never imports
      (`cli.go`'s package doc: cobra-independent). Matches `list.go`'s
      existing `writeListTable`/`writeListJSON` split exactly; this
      section's header grouped it under `internal/cli` by mistake when
      tasks.md was first written, before any of this existed to check
      against.
- [x] 2.3 Add JSON formatting (`json.NewEncoder`, per design.md) and
      verify a unit test checks the output round-trips through
      `encoding/json` back to the same entry set, including each entry's
      `id` field passed through unmodified. Same relocation as 2.2 above -
      done in `cmd/hush-hush-cli/audit_log.go`.

## 3. cmd/hush-hush-cli

- [x] 3.1 Add `audit_log.go` with the `audit-log` command: `--object`,
      `--actor`, `--caller`, `--since`, `--until`, `--format`, `--limit`
      flags, cobra `RunE` resolving config and calling `cli.AuditLog`,
      per the shape of `get.go`. Verify `hush-hush-cli audit-log --help`
      lists all seven flags with the descriptions from
      `specs/audit-log/spec.md`. **`--actor`, not the `--token` this task
      originally said** - design.md's new flag-naming Risk: `--token`
      collides with the root command's own persistent `--token` (the
      write-path bearer credential) in a way that silently breaks when
      the flag is given before the subcommand name, confirmed live with a
      throwaway cobra reproduction before writing this file. Done in
      `cmd/hush-hush-cli/audit_log.go`; `--help` manually checked and
      `TestAuditLogHelpListsEverySevenFlag` checks it mechanically.
      Table/JSON formatting (2.2/2.3 above) live here too.
- [x] 3.2 Parse `--since`/`--until` as RFC3339 (design.md) and verify a
      test covers a rejected non-RFC3339 value failing before any
      request is made. Done: `TestAuditLogRejectsAMalformedSinceBeforeAnyRequest`
      points `--server` at an unreachable address so a request attempt
      would fail differently, proving the RFC3339 parse error - not a
      connection error - is what actually surfaces.
- [x] 3.3 Reject a `--format` value other than `table`/`json` before any
      request is made, and verify a test covers it (spec's "Unknown
      format value rejected" scenario). Done:
      `TestAuditLogRejectsAnUnknownFormatBeforeAnyRequest`, same
      unreachable-server technique.
- [x] 3.4 Pass `--limit` through to `cli.AuditLog` (unset means "fetch
      everything", per design.md's revised paging decision) and verify a
      test covers a limit below and at/above the matching count (spec's
      two limit scenarios) — the actual paging and truncation live in
      `internal/client` (task 1.3); this layer only wires the flag
      through. Done: `TestAuditLogLimitBelowTheMatchingCountPrintsOnlyThatMany`
      and `TestAuditLogLimitAtOrAboveTheMatchingCountPrintsEveryEntry`.
- [x] 3.5 Verify a test covers zero filters returning everything the
      fake server has, subject only to `--limit` (spec's "No filters
      returns everything" scenario). Done: `TestAuditLogNoFiltersReturnsEverything`.

## 4. Docs

- [x] 4.1 Replace README's "query it directly" audit-log note with a
      section documenting the `audit-log` command and its flags, and
      verify the documented flag list matches `--help`'s actual output.
      README had no such note by the time this was implemented (checked:
      no "curl"/"audit" mention in it at all) - proposal.md's Why
      described the state before this repo's split from `hush-hush`
      (openspec/changes/split-cli-into-own-repo/), not this repo's
      current README. Added a "Usage" section example instead, covering
      `--object`/`--since`, `--actor`/`--format json`, and a short note on
      `--actor` vs `--caller`, `--limit`, and no `--follow`.
- [x] 4.2 Add the command to `INSTALL.md` or any generated man page
      (`cmd/hush-hush-cli/man.go`) covering per-command help, if that
      generation doesn't already pick it up automatically - verify by
      running whatever regenerates it and checking `audit-log` appears.
      Confirmed automatic: `go run ./cmd/hush-hush-cli man /tmp/man-out`
      produced `hush-hush-cli-audit-log.1` with no code change -
      `doc.GenManTree` walks `root`'s command tree, and `main.go`'s new
      `root.AddCommand(newAuditLogCmd())` is already part of it.
      `INSTALL.md` only references "a man page" generically, nothing
      per-command to update there.

## 5. Close out

- [x] 5.0 Not in the original plan: added
      `TestContainerAuditLogReturnsRecordedEntries` to
      `integration/container_test.go` - every other endpoint
      `internal/testserver`'s fake implements has a real-server fidelity
      check there (`rules/go-test.md`: "The container/integration layer
      uses testcontainers-go"), and this fake's own filtering/pagination
      logic (task 1.1) was written against `api/openapi.yaml` read
      directly, not against a live server, same risk design.md's paging
      Risk already flagged for the SDK. First run against a stale locally
      cached `ghcr.io/alrayyes/hush-hush:latest` (pulled the day before,
      predating alrayyes/hush-hush#214/#215) failed with `id` missing
      from the response entirely - alarming until `docker pull` refreshed
      the tag and it passed; not a real server bug, but worth recording
      since Docker doesn't refresh a `:latest` tag on its own and this
      cost real time to chase down.
- [x] 5.1 Run `go test ./...` and confirm it passes with the new tests
      included. Also ran with `-race` (clean) and
      `go test -tags=integration ./integration/...` against the real,
      freshly pulled `ghcr.io/alrayyes/hush-hush:latest` (all pass, see
      5.0).
- [x] 5.2 Open the pull request referencing `Closes #100`
      (alrayyes/hush-hush-cli#100), linking alrayyes/hush-hush#214,
      alrayyes/hush-hush#215, and alrayyes/hush-hush-go#74 (plus whatever
      issue tracks #215's SDK regen) as the unblocking work, per
      `skills/pull-request/`. Done: #103, merged by alrayyes as
      commit 983fb77. One more fix needed after opening it: CI's
      `nix-flake` job failed on a `vendorHash` mismatch in `flake.nix` -
      the hash pins the exact `go.mod`/`go.sum` dependency set, which the
      `hush-hush-go` v2.0.3 bump (task 0.3) changed. Updated to the value
      the failed build's own error reported
      (`rules/packaging.md`: "resolved by actually building, not
      guessed"), pushed as a follow-up commit, all checks green after.
- [x] 5.3 Archive this OpenSpec change (`openspec-archive-change`) once
      the pull request has merged. Doing this now.
