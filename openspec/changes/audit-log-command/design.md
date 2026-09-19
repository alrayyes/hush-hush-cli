# Design

## Context

See `proposal.md` for the "why". Relevant current state:

- `hush-hush-go`'s `Client.QueryAuditLog(ctx, AuditLogFilter)` already
  exists and returns `[]AuditLogEntry{Action, Caller *string, Ip string,
ObjectId, Timestamp time.Time}`, oldest-first, unpaginated. `AuditLogFilter`
  today has `ObjectID`, `Caller`, `From`, `To` — no token/actor field.
- `internal/client` (this repo) wraps the SDK behind its own types so
  `internal/cli` never imports `hush-hush-go` directly (this repo's
  `CLAUDE.md`).
- Every existing command (`get`, `delete`, `update`, `inject`) follows the
  same shape: a thin `cmd/hush-hush-cli/*.go` `RunE` that resolves config
  via `cliconfig`, then calls a same-named function in `internal/cli`
  that does the real work and returns a value or error for `RunE` to
  handle. None of them print a table today — this is the first command
  that needs one.
- Blocked on alrayyes/hush-hush#214 and alrayyes/hush-hush-go#74 (see
  proposal.md's Impact section) for the `--token` filter specifically.

## Goals / Non-Goals

**Goals:**

- Land a design and task breakdown ready to implement the instant
  hush-hush-go#74 ships, with no re-design needed at that point.
- Keep the command's shape consistent with the four existing commands.

**Non-Goals:**

- Implementing the command now — blocked (see proposal.md).
- Building a generic "table printer" abstraction for future commands to
  reuse. One `text/tabwriter` call, sized for this command's five columns,
  is enough; a shared helper can be extracted later if a second table-
  printing command shows up, not speculatively now.

## Decisions

- **Ship the full command in one piece, not the unblocked subset first.**
  `--object`/`--caller`/`--since`/`--until` could be built today against
  `hush-hush-go` as it stands; `--token` cannot. Considered shipping the
  first four now and adding `--token` in a follow-up once #214/#74 land.
  Rejected: that would ship a command whose flag surface visibly changes
  shape twice for one feature, and the "AND-combined filters" requirement
  (spec's first requirement) reads as one behavior, not two — testing it
  properly wants all five filters available together. The wait is for the
  server team's work, not this repo's; nothing here gets easier by
  splitting it.
- **`text/tabwriter` for the table format, `encoding/json` for
  `--format json`.** Both are standard library (`rules/go.md`'s "standard
  library first" — no third-party table or CLI-formatting library for a
  five-column, one-shot print). `json.NewEncoder(cmd.OutOrStdout())`
  writing the `[]AuditLogEntry`-shaped result directly, rather than a
  hand-rolled marshal, keeps the JSON output identical in shape to the
  server's own response.
- **`--limit` truncates client-side, after the full result set comes
  back.** The server's `/audit-log` has no pagination or limit parameter
  (`hush-hush-go`'s own doc comment on `QueryAuditLog`), so there's
  nothing to push the limit down to. This does mean a very large,
  loosely-filtered query still costs the same request/transfer regardless
  of `--limit` — acceptable for an operator-facing bounded query tool,
  not a hot path.
- **`--since`/`--until` parse as RFC3339, matching `AuditLogFilter.From`/
  `.To`'s `*time.Time` and the flag shape the peer session (working the
  server-side design) specified.** No looser format (bare dates, relative
  durations) — keeps flag parsing a one-line `time.Parse(time.RFC3339,
...)` with a clear error, consistent with how the server itself
  documents the endpoint's `from`/`to` params.
- **New `internal/cli` function, not a method added to an existing one.**
  Follows the existing per-command pattern (`cli.Get`, presumably
  `cli.Delete`/`cli.Update`/`cli.Inject` alongside it) rather than
  introducing a different shape for this one command.

## Risks / Trade-offs

- [Risk] The server's actual actor/token filter parameter name or
  semantics (once #214 ships) may not match the `--token` flag's assumed
  1:1 mapping to a single filter value → [Mitigation] `tasks.md` includes
  a task to re-check the merged `hush-hush` spec and `hush-hush-go`#74's
  actual `AuditLogFilter` field name/type before wiring `--token` through,
  rather than assuming the sketch in this design is final.
- [Risk] `AuditLogEntry.Caller` is `*string` (nil when the request had no
  `X-Caller` header) — a naive table print could show an empty column
  inconsistently → [Mitigation] `tasks.md` includes rendering a fixed
  placeholder (e.g. `-`) for a nil caller in table output; JSON output
  passes the value through as-is (`null`), unmodified.
- [Trade-off] No `--follow` mode means an operator watching for new
  activity has to re-run the command themselves → [Mitigation] Accepted
  per proposal.md's explicit non-goal, matching the server-side design
  doc's own precedent (`gh audit-log`, which is also bounded, not
  streaming).

## Open Questions

None — the two blockers in proposal.md's Impact section are hard
dependencies on other repos landing work, not open design questions;
nothing here is deferrable without changing the approach once they do
land (see the first Risk above for what specifically needs re-checking
against the real, merged spec at that point).
