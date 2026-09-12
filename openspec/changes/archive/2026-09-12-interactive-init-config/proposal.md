## Why

`hush-hush-cli init` writes an all-blank starter config file today, and no
command prompts or validates a missing value: `inject`/`update`/`delete`
silently proceed with an empty token and fail with an opaque server-side
error instead of a clear one. There's no way to configure the CLI without
hand-editing YAML or passing a flag — and a flag value for a secret lands in
shell history, the exact thing `inject`/`update` already avoid for the
sealed value itself. Tracked as
[alrayyes/hush-hush-cli#70](https://github.com/alrayyes/hush-hush-cli/issues/70).

## What Changes

- `init` prompts interactively for `server`, `token`, `caller`, `recipients`,
  `identity` when stdin is a TTY and `--yes` wasn't passed. Under `--yes` or
  with no TTY, it keeps writing today's blank template unchanged.
- For the two credential fields (`token`, `identity`), after the value is
  entered `init` offers a persistence choice: an OS keyring entry, a
  user-supplied retrieval command, the literal value in `config.yaml`, or
  skip (don't persist).
- New dependency `github.com/zalando/go-keyring` and a new hidden
  `hush-hush-cli config keyring-get <field>` subcommand — the generated
  `token_command`/`identity_command` when the keyring option is chosen.
- **New config field**: `identity_command` (config key `identity_command`,
  env `HUSH_HUSH_IDENTITY_COMMAND`), mirroring the existing `token_command` —
  today only `token` has a command sibling.
- The existing pre-command "no config file found, write one now?" nudge
  (`maybeOfferInit` in `cmd/hush-hush-cli/main.go`) now runs the same
  interactive flow `init` itself uses on confirmation, instead of writing a
  blank template.
- `inject`, `get`, `update`, `delete` gain no prompting of their own. A
  missing required value fails fast, and the error now also names
  `hush-hush-cli init` as a remedy alongside the flag/env var. This includes
  new token validation on `inject`/`update`/`delete`, which today has none.

## Capabilities

### New Capabilities

- `cli-config`: how `hush-hush-cli` resolves, persists and (now) interactively
  collects its connection settings — the config file, `init`, the
  flag/env/file precedence, and the literal-vs-command secret resolution for
  credential fields.

### Modified Capabilities

(none — `cli-config` did not exist as a captured spec before this change;
`openspec/specs/` holds only a `.gitkeep` today)

## Impact

- `cmd/hush-hush-cli/main.go`: `init`'s `RunE`, `maybeOfferInit`, `config()`,
  a new `config keyring-get` subcommand.
- `cmd/hush-hush-cli/get.go`: `identity`/`identity_command` resolution.
- `cmd/hush-hush-cli/inject.go`, `update.go`, `delete.go`: new token
  validation.
- `internal/cliconfig/`: prompting and secret-resolution helpers shared
  between `init` and the pre-command nudge.
- `go.mod`/`go.sum`: new pinned `github.com/zalando/go-keyring` dependency.
- `README.md`: Configuration section and flag/env/config table.
