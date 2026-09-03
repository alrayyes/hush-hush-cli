# Contributing

This file is for whoever changes this codebase. The [README](README.md) is
for whoever runs it.

## Getting set up

- **Go 1.27 or newer** — for editor/IDE tooling (`gopls`, `go build`/`go test`
  run by hand) and the container-integration test, which needs a real Go
  toolchain outside the hooks. Not required for the hooks themselves; see
  below.
- **[Docker](https://docs.docker.com/get-docker/)**, running. Every Go hook
  command (`golangci-lint fmt`/`run`, `go mod edit -fmt`/`tidy -diff`,
  `go test`, `actionlint`) runs through the `golang`/`golangci-lint` image
  `go.mod`'s own `go` directive and CI both pin, not whatever Go/
  golangci-lint happen to be on your `PATH` — a host toolchain drifting
  from the pinned image independently (a package manager updating one
  without the other) is a real failure mode this closes, not a
  hypothetical one.
- **[bun](https://bun.sh)** for the tooling that isn't Go — commitlint,
  Prettier, markdownlint, and the [lefthook](https://lefthook.dev) that runs
  the git hooks. There's a `package.json`, but nothing here is JavaScript; it
  exists only so those tools resolve and stay pinned.

One command installs the linters and the git hooks:

```sh
bun install
```

The `prepare` script runs `lefthook install` for you — an uninstalled hook
silently does nothing, which is worse than not having one.

## Everyday commands

The hooks and CI both run these through the pinned Docker image; running
them bare against your own `go`/`golangci-lint` is fine for quick
iteration, but only the Docker-wrapped form is the actual gate.

```sh
go build ./...
go vet ./...
go test ./...
go test -tags=integration ./integration/...  # needs Docker, boots the real hush-hush server
golangci-lint run
golangci-lint fmt          # the fixer; `run` stays the check

bun run format:check       # prettier --check, add --write to fix
bun run lint:md
```

## Where this came from

`hush-hush-cli` split out of [`hush-hush`](https://github.com/alrayyes/hush-hush),
which still owns the server and the secrets-object-store design this client
talks to. See [`openspec/changes/split-cli-into-own-repo/`](openspec/changes/split-cli-into-own-repo/)
for why the split happened. The split is done: the code
(`cmd/hush-hush-cli`, `internal/cli`, `internal/client`, `internal/seal`,
`internal/cliconfig`) lives only here now, `hush-hush` has removed its own
copy, and every packaging path (AUR, `.deb`/`.rpm`, Docker, `go install`)
ships from this repo's own release pipeline - see [INSTALL.md](INSTALL.md).
Nix (`hush-hush-cli#8`/`#9`) is what's left, not blocking anything.

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org/):
`type(scope): description`, types `feat`/`fix`/`docs`/`style`/`refactor`/
`perf`/`test`/`build`/`ci`/`chore`/`revert`. Subject under 50 characters,
lowercase, no trailing full stop. commitlint enforces the shape at
commit-msg and again in CI.

## Branching, review, and release

Every change goes through a pull request — nothing is pushed straight to
`main`. The one unavoidable exception is the commit that bootstrapped this
repo: there's no branch to open a pull request against before the first
commit exists.

The pull request **title** has to be a valid Conventional Commit too —
`pr-title.yml` checks it, since a squash merge defaults the squash commit's
message to the pull request title and commitlint only ever reads commit
objects.

[release-please](https://github.com/googleapis/release-please) reads the
Conventional Commits on `main` and keeps a release pull request open with
the next version and changelog entry; merging that one tags the release.
[goreleaser](https://goreleaser.com) then builds the binaries and packages
onto the release release-please just cut. Nobody picks a version by hand.
