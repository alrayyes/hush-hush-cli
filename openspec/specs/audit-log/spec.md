## Purpose

Lets an operator query hush-hush's audit trail from the CLI — who touched
which object or token, when, and how — instead of hand-crafting requests
against the server's `GET /audit-log` endpoint directly.

## Requirements

### Requirement: Filters combine with AND

The `audit-log` command SHALL accept `--object`, `--actor`, `--caller`,
`--since`, and `--until` flags, sending every flag that is set to the
server as a single request whose filters combine with AND, and SHALL
return only entries matching every filter given. `--actor` restricts to
entries authenticated by a specific verified actor - a token id, or the
admin account's own actor id - matching the server's own `actor` query
parameter; it is named `--actor`, not `--token`, to avoid colliding with
this CLI's existing global `--token` flag (the write-path bearer
credential), which means something entirely different.

#### Scenario: Single filter

- **WHEN** `audit-log --object secret-1` runs
- **THEN** only entries whose object ID is `secret-1` are printed

#### Scenario: Multiple filters combine with AND

- **WHEN** `audit-log --object secret-1 --caller ci-runner --since
2026-09-01T00:00:00Z` runs
- **THEN** only entries matching all three conditions are printed, not
  entries matching any one of them

#### Scenario: No filters returns everything

- **WHEN** `audit-log` runs with no filter flags
- **THEN** every entry the server returns is printed, subject only to
  `--limit`

### Requirement: No live/follow mode

The `audit-log` command SHALL make exactly one request per invocation and
exit once its result is printed. It SHALL NOT offer a `--follow` or tail
option that keeps the command running to stream new entries.

#### Scenario: Command exits after printing

- **WHEN** `audit-log` runs, with or without filters
- **THEN** the command prints its result and exits; it does not keep
  running or poll the server again

### Requirement: Table output by default

With no `--format` flag, `audit-log` SHALL print matching entries as a
human-readable table with resolved (local, non-epoch) timestamps, not raw
JSON.

#### Scenario: Default format is a table

- **WHEN** `audit-log` runs with no `--format` flag
- **THEN** output is a table whose timestamp column shows a
  human-readable date and time, not a Unix epoch or a bare RFC3339 string

### Requirement: JSON output with --format json

With `--format json`, `audit-log` SHALL print matching entries as JSON
that a downstream tool can parse, instead of the table.

#### Scenario: JSON format requested

- **WHEN** `audit-log --format json` runs
- **THEN** stdout is valid JSON representing the matching entries, and no
  table is printed

#### Scenario: Unknown format value rejected

- **WHEN** `audit-log --format xml` (or any value other than `table` or
  `json`) runs
- **THEN** the command fails with an error naming the allowed values and
  makes no request to the server

### Requirement: --limit caps the number of entries returned

With `--limit N` given, `audit-log` SHALL print at most N matching
entries, regardless of how many the server returns.

#### Scenario: Limit below the matching count

- **WHEN** the server returns more than N entries matching the given
  filters, and `--limit N` was given
- **THEN** only the first N are printed

#### Scenario: Limit at or above the matching count

- **WHEN** the server returns N or fewer entries matching the given
  filters, and `--limit N` was given
- **THEN** every matching entry is printed

### Requirement: Server errors surface as CLI errors, not raw output

When the server rejects an `audit-log` request, the command SHALL exit
non-zero and print the server's error message, without a stack trace or a
raw HTTP response dump.

#### Scenario: Server rejects an unknown filter value

- **WHEN** the server returns an error status for the given filters (for
  example, a malformed `--object` value)
- **THEN** the command exits non-zero and prints the server's error
  message on stderr
