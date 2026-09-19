# Tasks

## 0. Unblock

- [ ] 0.1 Confirm alrayyes/hush-hush#214 has merged to `hush-hush`'s main
      and its `api/openapi.yaml` carries the actor/token filter; confirm
      alrayyes/hush-hush-go#74 has shipped a released `hush-hush-go`
      version exposing it on `AuditLogFilter`. Do not start section 1
      before both are true.
- [ ] 0.2 Re-read the merged `hush-hush` spec's actual actor/token
      parameter name/type and `hush-hush-go`#74's resulting
      `AuditLogFilter` field, and update this file's remaining tasks (and
      `specs/audit-log/spec.md` if the flag mapping changed) if either
      differs from this change's assumption of a single `--token` value
      mapping to a single filter field (design.md's first Risk).
- [ ] 0.3 Bump this repo's `go.mod` to the released `hush-hush-go` version
      from 0.2, and verify `go build ./...` succeeds.

## 1. internal/client

- [ ] 1.1 Add an `AuditLogFilter`-equivalent type and `QueryAuditLog`
      method to `internal/client` wrapping the SDK's own (per this
      repo's CLAUDE.md: talk to hush-hush-go, not raw HTTP), and verify a
      unit test against `internal/testserver`'s fake covers object,
      caller, since, until and token filters combining with AND.
- [ ] 1.2 Verify a unit test covers the server error path (unknown
      filter value) mapping to `internal/client`'s existing sentinel/
      `ErrUnexpectedStatus` error shape, not a raw HTTP error.

## 2. internal/cli

- [ ] 2.1 Add `cli.AuditLog(ctx, cfg, filter)` following the shape of
      `cli.Get`, returning the matching entries (or an error), and verify
      a unit test covers it calling through to `internal/client` with the
      filters it was given.
- [ ] 2.2 Add table formatting (`text/tabwriter`, per design.md) with
      columns for timestamp (human-readable, local time), action, object
      ID, caller (`-` for nil, per design.md's second Risk), and IP, and
      verify a unit test checks the rendered output for a fixed sample
      entry set, including a nil-caller row.
- [ ] 2.3 Add JSON formatting (`json.NewEncoder`, per design.md) and
      verify a unit test checks the output round-trips through
      `encoding/json` back to the same entry set.

## 3. cmd/hush-hush-cli

- [ ] 3.1 Add `audit_log.go` with the `audit-log` command: `--object`,
      `--token`, `--caller`, `--since`, `--until`, `--format`, `--limit`
      flags, cobra `RunE` resolving config and calling `cli.AuditLog`,
      per the shape of `get.go`. Verify `hush-hush-cli audit-log --help`
      lists all seven flags with the descriptions from
      `specs/audit-log/spec.md`.
- [ ] 3.2 Parse `--since`/`--until` as RFC3339 (design.md) and verify a
      test covers a rejected non-RFC3339 value failing before any
      request is made.
- [ ] 3.3 Reject a `--format` value other than `table`/`json` before any
      request is made, and verify a test covers it (spec's "Unknown
      format value rejected" scenario).
- [ ] 3.4 Implement client-side `--limit` truncation on the returned
      entry slice (design.md - server has no limit param) and verify a
      test covers a limit below and at/above the matching count (spec's
      two limit scenarios).
- [ ] 3.5 Verify a test covers zero filters returning everything the
      fake server has, subject only to `--limit` (spec's "No filters
      returns everything" scenario).

## 4. Docs

- [ ] 4.1 Replace README's "query it directly" audit-log note with a
      section documenting the `audit-log` command and its flags, and
      verify the documented flag list matches `--help`'s actual output.
- [ ] 4.2 Add the command to `INSTALL.md` or any generated man page
      (`cmd/hush-hush-cli/man.go`) covering per-command help, if that
      generation doesn't already pick it up automatically - verify by
      running whatever regenerates it and checking `audit-log` appears.

## 5. Close out

- [ ] 5.1 Run `go test ./...` and confirm it passes with the new tests
      included.
- [ ] 5.2 Open the pull request referencing `Closes #100`
      (alrayyes/hush-hush-cli#100), linking alrayyes/hush-hush#214 and
      alrayyes/hush-hush-go#74 as the unblocking work, per
      `skills/pull-request/`.
- [ ] 5.3 Archive this OpenSpec change (`openspec-archive-change`) once
      the pull request has merged.
