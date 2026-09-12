## Context

See proposal.md - Why. `internal/cliconfig` already owns "the config-file
convention every hush-hush binary follows" (its own doc comment) — the path
resolution (`Path`), existence check (`Exists`), the yes/no `Confirm`
prompt, and the `ShouldWriteStarter` decision used by `maybeOfferInit`.
`cmd/hush-hush-cli/main.go` owns cobra/viper wiring, the `starterConfig`
template, `writeStarterConfig`, and `config()`'s token/token_command
resolution. `identity` is resolved separately, directly in `get.go`, and has
no `_command` support today.

## Goals / Non-Goals

**Goals:**

- Prompting and secret-resolution logic reusable between `init`'s own `RunE`
  and `maybeOfferInit`'s nudge, with no duplicated flow.
- `token` and `identity` resolve through the same literal-or-command
  mechanism, sharing one helper instead of two copies of `runSecretCommand`
  call sites.
- Keep every existing non-interactive code path (flags/env/file precedence,
  `--yes`, no-TTY, `--force`) byte-for-byte unchanged.

**Non-Goals:**

- Editing an existing config file field-by-field. `--force` still means
  "start over," not "patch one field."
- Any change to config file permissions, or to precedence ordering.
- Prompting inside `inject`/`get`/`update`/`delete` themselves.

## Decisions

### Keyring library: `github.com/zalando/go-keyring`

Small API surface (`Set`/`Get`/`Delete` against macOS Keychain, Windows
Credential Manager, and Linux Secret Service/kwallet over D-Bus), no
transitive dependency tree to speak of. Considered `99designs/keyring`,
which supports more backends (file-based, pass, several cloud KMSs) but is
heavier and file-backend-capable in a way that would blur the line with the
"literal value" and "user-supplied command" choices already in this design
— `zalando/go-keyring`'s narrower "OS-native secret store only" scope maps
directly onto the spec's "operating system's credential store" option
without needing to also decide which of its extra backends to expose.

### Hidden `config keyring-get <field>` subcommand, not a flag or env var

The generated `token_command`/`identity_command` has to be an executable
command string (per the existing `_command` convention — see `rules/cli.md`
"Secrets get a command option, not just a value"), so retrieving from the
keyring needs _something_ invokable from a shell. Reusing the CLI's own
binary (`hush-hush-cli config keyring-get token`) needs no new binary,
installs nowhere extra, and works identically wherever the CLI itself is
installed (AUR, Docker, `go install`, ...). `cmd.Hidden = true` on both the
`config` group and `keyring-get` keeps it out of `--help` and the generated
man pages (`newManCmd` walks the visible command tree) — it's an
implementation detail of the `_command` mechanism, not a command a user
would type by hand outside that context. Writing to the keyring needs no
equivalent subcommand: `init` runs in-process and can call `keyring.Set`
directly.

### Shared secret-resolution helper in `internal/cliconfig`

`config()` in `main.go` already inlines "if `token_command` is set, run it
via `runSecretCommand` and use its output, else use the literal value."
`identity_command` needs the identical shape. Rather than duplicate that
five-line block, both call one new `cliconfig.ResolveSecret(literal,
command string) (string, error)` — moving `runSecretCommand` itself into
`internal/cliconfig` alongside it, since it stops being main-package-only
once a second caller needs it. `get.go`'s `RunE` calls it for
`identity`/`identity_command` the same way `config()` calls it for
`token`/`token_command`.

### Interactive flow lives in `internal/cliconfig`, not `cmd/hush-hush-cli`

`rules/go.md`'s "keep RunE a thin shell" and the module-layout guidance
against non-cobra logic in `cmd/`. A new `cliconfig.PromptConfig(in
io.Reader, out io.Writer, existing cliconfig.Values) (cliconfig.Values,
error)`-shaped function (exact signature decided during implementation)
walks the five fields, using masked reads (`golang.org/x/term.ReadPassword`)
for `token`/`identity` and plain reads (extending the existing
`bufio.Scanner` pattern `Confirm` already uses) for the rest, then a small
persistence-choice prompt for the two credential fields. `init`'s `RunE`
and `maybeOfferInit` both call it and then render the result into
`config.yaml` — the YAML-rendering step stays in `main.go` next to
`writeStarterConfig`/`starterConfig`, since `cliconfig` has no YAML
dependency today and this doesn't need to introduce one to the package that
also has to run inside `config keyring-get`.

### Token validation added at the `cli.Config` boundary

`inject`/`update`/`delete` already call `config()` then `cfg.Validate()`
(`internal/cli/cli.go`). `Validate()` today only requires `Server`. Rather
than add three near-identical "if cfg.Token == "" { return errNoToken }"
checks in `cmd/hush-hush-cli/{inject,update,delete}.go`, add a
`RequireToken bool` field callers set before validating, since `get` is the
one command that must NOT require it — mirrors how `Config` already
documents that asymmetry in its own doc comment ("needed by inject, update,
and delete, ignored by get").

## Risks / Trade-offs

- **New dependency surface** (`zalando/go-keyring`) that itself shells out
  to platform secret-storage tools/daemons (`secret-tool`/D-Bus on Linux) →
  mitigated by it being one optional path among four; choosing "literal
  value" or "user-supplied command" at the prompt needs it not at all, and
  `config keyring-get` only ever runs when a config file explicitly opts
  into it.
- **Headless Linux with no Secret Service running** (a bare CI container, an
  SSH-only box) makes the keyring option fail at _use_ time, not at prompt
  time → mitigated by keeping it a prompt choice, never a default; the
  prompt happens on a real TTY, where a user picking it can reasonably be
  expected to be on a desktop session that has one running.
- **`golang.org/x/term.ReadPassword` swallows the newline the user types**,
  same as any masked-password prompt → the existing TTY-check code already
  imports `x/term`, so this is a new call, not a new dependency; documented
  behavior of that package, not a risk to design around.

## Open Questions

None — the scope and every choice above were settled in the linked issue
and its preceding discussion before this design was written.
