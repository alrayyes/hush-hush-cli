# hush-hush-cli

Built from `~/.config/claude/CLAUDE.md` and `~/.config/claude/rules/*.md` —
read those for the "why" behind everything below. This file only says
what's specific to this repo.

## What this is

The client for [hush-hush](https://github.com/alrayyes/hush-hush)'s
secrets object store. Split out of that repo into its own — see
[`openspec/changes/split-cli-into-own-repo/`](openspec/changes/split-cli-into-own-repo/)
for why and what moved. The split is done: `hush-hush` has removed its own
copy of the CLI and points here instead.

## Gotchas

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
- **Connect to `hush-hush` through the `hush-hush-go` SDK, not raw HTTP** -
  Ryan's own standing preference. `internal/client` wraps it for exactly
  this reason; a new test or tool talking to a real or containerized
  server should go through it too. The one deliberate exception is
  `hush-hush`'s own `integration/container_test.go` on the _server_ side,
  which proves the packaged artifact over the wire - not something to
  retrofit here.
- **Two Dockerfiles, on purpose.** `Dockerfile` is the from-source build a
  contributor runs locally; `Dockerfile.release` only `COPY`s the binary
  goreleaser already cross-compiled (`go-releases.md`'s reproducibility
  tradeoff). Never run `Dockerfile.release` by hand - it has no source to
  build from.
- **Packaging is done**: AUR (`hush-hush-cli-bin`, repointed at this repo),
  `.deb`/`.rpm`, `go install`, and a Docker image
  (`ghcr.io/alrayyes/hush-hush-cli`) all ship from this repo's own release
  pipeline. See [INSTALL.md](INSTALL.md). What's left is Nix
  (`hush-hush-cli#8`/`#9`) - not blocking anything.
