## Why

`hush-hush-cli` and the `hush-hush` server share one repo, one Go module,
and one release version today. Every test in `internal/cli` and
`cmd/hush-hush-cli` boots the real server in-process (`internal/api.NewMux`
via `httptest`) to exercise the client, and a CLI-only flag addition bumps
the same release-please version and triggers the same goreleaser job as a
server change, republishing the server's GHCR image alongside the CLI's
AUR/deb/rpm/Nix packages either way. Splitting removes that coupling and
gives `hush-hush-cli` unambiguous `cli.md` compliance (`init`, a config
file, XDG paths) with no "well the server does this too" argument muddying
`hush-hush#141`'s open question about the server's own shape.

## What Changes

- New repo `github.com/alrayyes/hush-hush-cli`, bootstrapped with its own
  CI, release-please/goreleaser release pipeline, and OpenSpec tracking
  (this change).
- `cmd/hush-hush-cli`, `internal/cli`, `internal/client`, `internal/seal`,
  and `internal/cliconfig` move here from `hush-hush`, verbatim in behavior
  - same commands, same flags, same config precedence. **Not a rewrite.**
- **BREAKING** (packaging only, not CLI behavior): the AUR package
  (`hush-hush-cli-bin`), `.deb`/`.rpm` release assets, and the Nix flake
  move to this repo's own release pipeline. Anyone installing from
  `hush-hush`'s release page or its AUR package needs to repoint at this
  repo once the migration ships.
- `hush-hush` removes its copy of the above once this repo's version is
  verified working end to end - tracked as this change's own task, not
  assumed done just because this repo exists.
- Deferred, tracked as follow-up tasks rather than blocking the move: a
  Docker image for the CLI (`hush-hush#152`), Nix flake CI verification
  (`hush-hush#139`), `flake.lock` update automation (`hush-hush#153`) - all
  closed in `hush-hush` pointing here.

## Capabilities

No capability changes. The CLI's user-facing behavior - what `inject`/
`get`/`update`/`delete`/`init` do, `secrets-object-store`'s own
`specs/cli/spec.md` in the `hush-hush` repo - stays exactly what it is;
this change moves where the code implementing it lives, not what it does.
`skip_specs: true` set in `.openspec.yaml` accordingly.

## Impact

- **This repo**: everything under `cmd/`, `internal/`, plus packaging
  config (`.goreleaser.yml`, PKGBUILD, `flake.nix`, `INSTALL.md`) grows in
  from `hush-hush` as migration tasks land.
- **`hush-hush`**: `cmd/hush-hush-cli`, `internal/cli`, `internal/client`,
  `internal/seal`, `internal/cliconfig`, the CLI's AUR/deb/rpm/Nix
  packaging, and its own copy of the pact consumer test all get removed
  once this repo's replacement is verified working. `internal/api` (the
  pact provider side) and its own test suite stay - only the consumer
  moves.
- **Consumers of the CLI**: a repoint from `hush-hush`'s AUR package /
  release assets to this repo's own, once it ships a first real release.
  No command, flag, or config-file behavior changes for them.
- **`hush-hush#141`** (does the server binary belong with `cli.md`'s
  backend-service carve-out): not resolved by this change - stays a
  `hush-hush` decision about `cmd/hush-hush`'s own `init`/config/`token`
  subcommands, unaffected by where the CLI lives.
