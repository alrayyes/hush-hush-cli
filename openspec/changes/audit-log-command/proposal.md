# Proposal

## Why

hush-hush's `GET /audit-log` is the only way to see who touched a secret
object and when, and this CLI has no wrapper for it at all — the README
tells an operator to query it directly with `curl`. That's a real gap for
day-to-day use, and it's about to get more useful to close: `hush-hush`'s
`web-ui` OpenSpec change (alrayyes/hush-hush#198, design.md's "The CLI's
own audit-log command" note) is adding verified actor/token attribution
to that endpoint, and its design explicitly expects this CLI to gain a
command against it rather than leaving that only reachable from the new
web UI. Tracked here as alrayyes/hush-hush-cli#100.

## What Changes

- New `hush-hush-cli audit-log` command with `--object`, `--token`,
  `--caller`, `--since`, `--until`, `--format table|json`, and `--limit`
  flags. Filters combine with AND, mirroring the server's `object_id`/
  `caller`/`from`/`to` query params (already supported by
  `hush-hush-go`'s `QueryAuditLog`/`AuditLogFilter`) plus the new
  actor/token filter the server is adding. No `--follow`/tail mode — this
  is a bounded query, not a live stream, matching `gh`'s own `audit-log`
  command shape (the server-side design doc's own cited precedent).
- Default output is a human-readable table with resolved (not raw
  epoch/RFC3339) timestamps; `--format json` prints the entries as JSON.
- `--limit N` truncates the printed/returned set to at most N entries
  client-side (the server's `/audit-log` has no pagination parameters —
  `QueryAuditLog` always returns the full matching result set in one
  call).

## Capabilities

### New Capabilities

- `audit-log`: the CLI's `audit-log` command — its flags, how they map to
  server-side filters, and its two output formats.

### Modified Capabilities

None. `cli-config` (the only existing spec) covers configuration
resolution, not commands; this doesn't change how config is loaded.

## Impact

- New file `cmd/hush-hush-cli/audit_log.go` (+ test), following the shape
  of the existing `get.go`/`delete.go` commands: a thin `RunE` that
  resolves config and calls into a new `internal/cli` function.
- New function in `internal/cli` (alongside `cli.Get`, etc.) that maps CLI
  flags to `internal/client`'s wrapper over `hushhush.AuditLogFilter`, and
  formats the result as a table or JSON.
- **Blocked on two upstream changes**, in order:
  1. alrayyes/hush-hush#214 (actor/token bearer-attribution + the
     `GET /audit-log` actor filter) — not started as of this proposal;
     the design revision (#216) and schema-only change (#217) are ahead
     of it in the queue and not yet merged either.
  2. alrayyes/hush-hush-go#74 (regenerating this repo's SDK dependency
     once #214's spec lands) — `hush-hush-go` doesn't expose an
     actor/token filter today; only `ObjectID`/`Caller`/`From`/`To` exist
     on `AuditLogFilter`.

  The `--object`/`--caller`/`--since`/`--until` subset of this command
  could technically be built against `hush-hush-go`'s SDK as it stands
  today. `design.md` records why this proposal ships the full command in
  one piece instead of splitting that subset out early.

- README gains an `audit-log` section replacing the current "query it
  directly" note once this ships.
