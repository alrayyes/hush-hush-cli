# hush-hush-cli

Built from `~/.config/claude/CLAUDE.md` and `~/.config/claude/rules/*.md` —
read those for the "why" behind everything below. This file only says
what's specific to this repo.

## What this is

The client for [hush-hush](https://github.com/alrayyes/hush-hush)'s
secrets object store, split out of that repo into its own — see
[`openspec/changes/split-cli-into-own-repo/`](openspec/changes/split-cli-into-own-repo/)
for why and what's moving, and [issue #1](https://github.com/alrayyes/hush-hush-cli/issues/1)
for the migration's own tracking.

## Gotchas

- **This repo is a bootstrap shell until the migration lands.** No
  `cmd/`/`internal/` yet — those packages (`cmd/hush-hush-cli`,
  `internal/cli`, `internal/client`, `internal/seal`, `internal/cliconfig`)
  still live in `hush-hush` and move here as tracked tasks.
- **`LICENSE` is GPL-3.0**, matching `hush-hush`'s own choice — code
  extracted from a GPL-3.0 repo stays GPL-3.0.
- **No Vale/ltex prose-quality tier**, unlike `hush-hush` itself — Markdown
  gets Prettier + markdownlint (mechanics) only, matching the leaner
  `hush-hush-{go,python,node,php}` sibling repos rather than `hush-hush`'s
  own heavier tooling.
- **No pact/contract-testing job yet.** `hush-hush`'s own
  `internal/client`/`internal/api` pact suite assumed same-repo access to
  the provider, which the split breaks — the replacement strategy (mock
  server against the pinned spec, `hush-hush-go`'s Prism-based `contract`
  job, or dropping it in favor of integration/e2e tests) is an open
  question in the migration's `design.md`, not decided yet.
- **No Docker image, PKGBUILD, or Nix flake yet** — each was a `hush-hush`
  ticket before the split (`hush-hush#152`/`#153`/`#139`) and moves here
  once there's real code and a first release to package.
