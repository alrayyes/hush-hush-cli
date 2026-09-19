# Design

## Context

See `proposal.md` for the "why". Relevant current state:

- `hush-hush-go`'s `Client.QueryAuditLog(ctx, AuditLogFilter)` today
  returns `[]AuditLogEntry{Action, Caller *string, Ip string, ObjectId,
Timestamp time.Time}`, oldest-first, matching the server's shape before
  its two most recent changes. `AuditLogFilter` today has `ObjectID`,
  `Caller`, `From`, `To` — no actor/token or pagination field.
- The server's own `GET /audit-log` (`api/openapi.yaml`) has since changed
  twice, neither yet reflected in `hush-hush-go`: alrayyes/hush-hush#214
  (merged 2026-09-19) added an `actor` query filter (a token id or the
  admin account's `actor_id` — verified, unlike `caller`) plus
  `actor_type`/`actor_id` response fields; alrayyes/hush-hush#215 (merged
  2026-09-19) added real cursor pagination — optional `after` (int64, the
  previous page's last entry's own new `id`) and optional `limit`
  (default 50, max 500), response still a plain array, oldest-first, one
  page per call. `hush-hush-go#74` tracks the actor/token regen; the
  pagination fields need the same kind of regen, no tracking issue for
  that half exists yet as of this update.
- `internal/client` (this repo) wraps the SDK behind its own types so
  `internal/cli` never imports `hush-hush-go` directly (this repo's
  `CLAUDE.md`).
- Every existing command (`get`, `delete`, `update`, `inject`) follows the
  same shape: a thin `cmd/hush-hush-cli/*.go` `RunE` that resolves config
  via `cliconfig`, then calls a same-named function in `internal/cli`
  that does the real work and returns a value or error for `RunE` to
  handle. None of them print a table today — this is the first command
  that needs one.
- Blocked on alrayyes/hush-hush-go regenerating against both
  alrayyes/hush-hush#214 (tracked as hush-hush-go#74) and
  alrayyes/hush-hush#215 (no tracking issue yet) — see proposal.md's
  Impact section.

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
- A `--page-size` flag or any other way to change how many entries
  `internal/client` requests per page. `--limit` is the only user-facing
  control; the server's own default/max (50/500) is what each page
  request uses internally.

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
- **`--limit` drives the server's own `limit`/`after` cursor pagination,
  not a client-side truncation of one unpaginated response.**
  alrayyes/hush-hush#215 (merged 2026-09-19, after this decision was first
  written against an unpaginated endpoint) gave `/audit-log` real paging:
  `limit` (default 50, max 500 per page) and `after` (the previous page's
  last entry's own new `id` field). `internal/client`'s wrapper passes
  `--limit` straight through as the first page's `limit` (capped at 500)
  when it fits in one page; when it doesn't — `--limit` above 500, or no
  `--limit` at all, per the "no filters returns everything" requirement —
  it loops, requesting the next page with `after` set to the prior page's
  last entry's `id`, until either the requested count is reached or a
  short page (fewer entries than requested) signals the server has
  nothing left. This replaces the fetch-everything-then-truncate approach
  the first draft used, back when the server had no pagination
  parameters at all; the user-visible contract (spec.md's limit
  scenarios) is unchanged.
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
- [Risk] The paging decision above is written against `api/openapi.yaml`
  as read directly, not against `hush-hush-go`'s regenerated SDK — the
  SDK's actual `After`/`Limit` field names, types, and short-page-means-
  done semantics could differ once a regen exposes them →
  [Mitigation] `tasks.md`'s unblock section now checks this alongside the
  existing actor/token re-check, before task 1 starts.

## Open Questions

None — the blockers in proposal.md's Impact section are hard dependencies
on other repos landing work, not open design questions; nothing here is
deferrable without changing the approach once they do land (see the Risks
above for what specifically needs re-checking against the real,
regenerated SDK at that point).
