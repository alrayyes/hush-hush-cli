# Contributing

This file is for whoever changes this codebase. The [README](README.md) is
for whoever runs it.

## Getting set up

- **Go 1.27 or newer.**
- **[bun](https://bun.sh)** for the tooling that isn't Go — commitlint,
  Prettier, markdownlint, and the [lefthook](https://lefthook.dev) that runs
  the git hooks. There's a `package.json`, but nothing here is JavaScript; it
  exists only so those tools resolve and stay pinned.
- **[golangci-lint](https://golangci-lint.run) v2.13.1**, which the
  pre-commit hook runs from your `PATH` while CI runs it pinned. Install
  that version rather than whichever is current: when the two disagree, the
  hook passes and the pipeline fails, and the reason isn't obvious from the
  failure.

One command installs the linters and the git hooks:

```sh
bun install
```

The `prepare` script runs `lefthook install` for you — an uninstalled hook
silently does nothing, which is worse than not having one.

## Everyday commands

```sh
go build ./...
go vet ./...
go test ./...
golangci-lint run
golangci-lint fmt          # the fixer; `run` stays the check

bun run format:check       # prettier --check, add --write to fix
bun run lint:md
```

## Where this came from

`hush-hush-cli` split out of [`hush-hush`](https://github.com/alrayyes/hush-hush),
which still owns the server and the secrets-object-store design this client
talks to. See [`openspec/changes/split-cli-into-own-repo/`](openspec/changes/split-cli-into-own-repo/)
for why the split happened and what's moving. Until that change's tasks are
done, this repo is a bootstrap shell — the actual `cmd/hush-hush-cli`,
`internal/cli`, `internal/client`, `internal/seal` and `internal/cliconfig`
packages still live in `hush-hush` and haven't moved yet.

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org/):
`type(scope): description`, types `feat`/`fix`/`docs`/`style`/`refactor`/
`perf`/`test`/`build`/`ci`/`chore`/`revert`. Subject under 50 characters,
lowercase, no trailing full stop. commitlint enforces the shape at
commit-msg and again in CI.

## Branching, review, and release

Every change goes through a pull request — nothing is pushed straight to
`main`, including the commit that bootstrapped this repo.

The pull request **title** has to be a valid Conventional Commit too —
`pr-title.yml` checks it, since a squash merge defaults the squash commit's
message to the pull request title and commitlint only ever reads commit
objects.

[release-please](https://github.com/googleapis/release-please) reads the
Conventional Commits on `main` and keeps a release pull request open with
the next version and changelog entry; merging that one tags the release.
[goreleaser](https://goreleaser.com) then builds the binaries and packages
onto the release release-please just cut. Nobody picks a version by hand.
