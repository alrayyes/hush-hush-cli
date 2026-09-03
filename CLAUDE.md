# hush-hush-cli

Built from `~/.config/claude/CLAUDE.md` and `~/.config/claude/rules/*.md` —
read those for the "why" behind everything below. This file only says
what's specific to this repo.

## What this is

The client for [hush-hush](https://github.com/alrayyes/hush-hush)'s
secrets object store, split out of that repo into its own — see
[`openspec/changes/split-cli-into-own-repo/`](openspec/changes/split-cli-into-own-repo/)
for why and what's moving, and [issue #6](https://github.com/alrayyes/hush-hush-cli/issues/6)
for the migration's own tracking.

## Gotchas

- **The code has landed** (`cmd/hush-hush-cli`, `internal/cli`,
  `internal/client`, `internal/seal`, `internal/cliconfig`), verified
  against a real running `hush-hush` server — see
  [`openspec/changes/split-cli-into-own-repo/`](openspec/changes/split-cli-into-own-repo/)
  for what's still open (first release, packaging, `hush-hush` removing
  its own copy).
- **`LICENSE` is GPL-3.0**, matching `hush-hush`'s own choice — code
  extracted from a GPL-3.0 repo stays GPL-3.0.
- **No Vale/ltex prose-quality tier**, unlike `hush-hush` itself — Markdown
  gets Prettier + markdownlint (mechanics) only, matching the leaner
  `hush-hush-{go,python,node,php}` sibling repos rather than `hush-hush`'s
  own heavier tooling.
- **No pact/contract-testing job.** `hush-hush`'s own `internal/client`/
  `internal/api` pact suite assumed same-repo access to the provider,
  which the split breaks. This repo drops the consumer side rather than
  standing up a broker; `internal/testserver`'s hand-written stateful fake
  (not Prism — design.md's resolved Open Question) covers the
  create/get/update/delete round trip in `go test ./...` instead. Pact
  stays on `hush-hush`'s provider side only.
- **`integration/` is a separate, build-tagged (`integration`) test tier**
  that boots the real, published `ghcr.io/alrayyes/hush-hush` image via
  `testcontainers-go` and checks `internal/testserver`'s fake against it -
  not part of `go test ./...`, its own CI job and `pre-push` hook instead
  (`go test -tags=integration ./integration/...`), since it needs a
  Docker daemon and network access.
- **No Docker image, PKGBUILD, or Nix flake yet** — each was a `hush-hush`
  ticket before the split (`hush-hush#152`/`#153`/`#139`) and moves here
  once there's a first release to package (tasks.md §4).
