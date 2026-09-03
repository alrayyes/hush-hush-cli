## Context

See proposal.md for why. The code moving is small and well-understood -
`internal/seal` (~145 lines incl. tests) and `internal/client` (~324 lines
incl. tests) are genuinely tiny - but `internal/cli` and
`cmd/hush-hush-cli`'s entire test suite currently boots the real server
in-process (`internal/api.NewMux` via `httptest`, a `newTestServer(t)`
helper every test file calls) rather than mocking it or hitting a deployed
instance. That coupling, not the code volume, is what a same-repo layout
was actually buying, and it's what this design has to replace.

## Goals / Non-Goals

**Goals:**

- Move `cmd/hush-hush-cli`, `internal/cli`, `internal/client`,
  `internal/seal`, `internal/cliconfig` with no behavior change - same
  commands, flags, config precedence, output.
- Keep both repos' CI green throughout - `hush-hush`'s copy stays working
  until this repo's replacement is verified, not removed preemptively.
- Land this repo's own version of every 2026-09-03 rules-hardening fix
  `hush-hush` already went through (`Validate()` on config, `-diff` on
  lockfiles, pinned toolchain versions, etc.) rather than reintroducing
  gaps already closed once.

**Non-Goals:**

- A rewrite, refactor, or behavior change to any command. This is a
  location move.
- Packaging (Docker image, AUR, deb/rpm, Nix) in the first pass - tracked
  as follow-up tasks once there's a real release to package.
- Resolving `hush-hush#141` (the server's own `init`/config shape) - that's
  a `hush-hush`-side decision, unaffected by this repo existing.

## Decisions

- **Verbatim move, not a rewrite.** Every file's logic ports as-is; only
  import paths (`github.com/alrayyes/hush-hush/internal/...` ->
  `github.com/alrayyes/hush-hush-cli/internal/...`) and the module's own
  dependency on `internal/api` (gone, see testing below) change. Lower risk
  than "improve while moving" - any real improvement (the interface
  question `hush-hush#150` raised for `internal/cli.Config.newClient`, say)
  is its own follow-up task, not bundled into the move itself.
- **Testing: replace `httptest` against the real server with a fake
  `hush-hush-go` HTTP transport, not a mock server or a live deployment.**
  `internal/client` already wraps `hush-hush-go` (the generated SDK, this
  repo's actual transport) rather than talking to `internal/api` directly -
  the test helper booted a real server only because both lived in the same
  binary and it was cheap to. Once split, the natural fake is an
  `httptest.Server` serving canned responses matching `api/openapi.yaml`
  (Prism, the same tool `hush-hush-go`'s own `contract` CI job already
  uses against a pinned spec submodule, is the concrete option) - reach for
  a real deployed `hush-hush` (docker-compose in CI, the shape
  `hush-hush`'s own `container-integration` test already proves works)
  only if Prism's fidelity turns out to be insufficient once this is
  actually built. Decided at task-implementation time, not here - see Open
  Questions.
- **Pact contract testing: drop from this repo, keep on `hush-hush`'s
  provider side only, until proven otherwise.** `hush-hush`'s pact suite
  was consumer (`internal/client`) and provider (`internal/api`) in one
  repo with a local pact file, no broker. Split, the consumer would need
  to publish a pact file somewhere the provider's own CI can read it - a
  broker, or an artifact hand-off between two repos' CI runs - real
  infrastructure this change doesn't add. `hush-hush`'s provider test and
  `internal/api` stay; this repo starts without a consumer pact suite and
  relies on the fake-transport tests above plus `hush-hush-go`'s own tests
  against the real spec for coverage. Revisit if a real regression this
  gap would have caught actually happens.
- **Version starts at 0.0.0, not carried over from `hush-hush`'s current
  version.** `releases.md`'s initial-version note: release-please's own
  manifest starts empty and computes the first real version from commit
  history, same as every other repo in this family did. `hush-hush-cli`
  the binary keeps working at whatever version users already have; this is
  about this _repo's_ tag history, not the binary's user-visible behavior.
- **License: GPL-3.0, matching `hush-hush`.** Code extracted from a
  GPL-3.0 repo stays GPL-3.0 - not a fresh decision to make per
  `hush-hush`'s own `CLAUDE.md` note about the scaffold's own
  license-per-project convention, since this isn't a fresh scaffold, it's
  a literal split of existing GPL-3.0 code.
- **No Vale/ltex prose tier, unlike `hush-hush`.** Matches the leaner
  `hush-hush-{go,python,node,php}` sibling repos - Prettier + markdownlint
  only. A CLI repo with a README/CONTRIBUTING/CLAUDE.md doesn't carry the
  same prose surface `hush-hush`'s own OpenAPI-adjacent docs do.
- **`lefthook.yml`'s Go hooks run against the host toolchain, not
  Docker-wrapped, matching `hush-hush`'s own current (ticketed, not yet
  fixed) state rather than getting ahead of it here.** Tracked as its own
  follow-up task rather than solved uniquely in this repo - the fix, once
  written, should look the same in both.

## Risks / Trade-offs

- **[Risk] The fake-transport test replacement turns out less faithful
  than the real in-process server was, missing a regression the old setup
  would have caught.** → Mitigation: keep `hush-hush`'s own copy of
  `internal/cli`'s tests running (not deleted) until this repo's
  replacement suite is reviewed and its coverage compared line-for-line,
  not just "tests pass."
- **[Risk] A consumer following `hush-hush`'s README/AUR package after the
  packaging migration ships gets a stale, unmaintained copy.** →
  Mitigation: `hush-hush` removes its own copy (README section, AUR
  package, release assets) in the same pass that verifies this repo's
  replacement works, per proposal.md's "Impact" - not left dangling as a
  second, decaying install path.
- **[Risk] Splitting `internal/seal`/`internal/client` out doesn't reduce
  the interface question `hush-hush#150` raised for
  `internal/cli.Config.newClient` on its own - it's still there, just in a
  different repo.** → Mitigation: explicitly not solved by this change
  (Non-Goals); left as a follow-up task, since the split may change what
  the right answer looks like (no shared module with `internal/api`
  anymore) once someone's actually looking at it.

## Migration Plan

1. Bootstrap this repo (done by this change's own tasks: CI, release
   pipeline, OpenSpec).
2. Move the five packages verbatim, replacing the `httptest`-against-real-
   server test helper per the testing decision above.
3. Get this repo's `go build/vet/test`, `golangci-lint run`, and a real
   `hush-hush-cli` binary built from it to parity with what `hush-hush`
   ships today - manually verified against a running `hush-hush` server,
   not just green CI.
4. Cut this repo's first real release (packaging - AUR/deb/rpm/Nix - as
   its own follow-up tasks, not blocking this one).
5. Only then: remove the five packages, the CLI's AUR/deb/rpm/Nix
   packaging, and the pact consumer test from `hush-hush`, and update its
   README to point here instead of documenting the CLI itself.

No rollback beyond "don't do step 5 until step 3 is actually verified" -
nothing user-facing changes until the old copy is removed, so there's no
mid-migration state a user can be stuck in.

## Open Questions

- ~~**Prism-mock vs. real-server-in-CI for this repo's own test suite**~~
  **Resolved: neither.** The moved tests (`TestGetDecryptsTheStoredValue`,
  `TestInjectCreatesAnObjectTheMatchingIdentityCanDecrypt`, and others)
  depend on real stateful create-then-get round trips and specific
  401/404/409 semantics against live state - a Prism spec-mock only
  returns canned per-operation examples with no cross-request state, so it
  can't reproduce that, and real-server-in-CI reintroduces the exact
  cross-repo coupling this migration removes. `internal/testserver` is a
  small hand-written in-memory stateful fake implementing the four
  `/objects` endpoints internal/client actually calls (matching
  `internal/api`'s handler contract in status codes and response shape,
  without depending on that package) - it satisfies "replace the
  in-process real server" without losing the round-trip fidelity the
  existing tests rely on.

  This resolves what replaces `newTestServer(t)` for `internal/cli`'s and
  `cmd/hush-hush-cli`'s own fast tests - it doesn't mean this repo has no
  real-server tier at all. Ryan asked directly for one after this decision
  landed: `integration/` (build-tagged `integration`, its own CI job and
  `pre-push` hook, matching `hush-hush`'s own `container-integration`
  test's shape) boots the real, published `ghcr.io/alrayyes/hush-hush`
  image via `testcontainers-go` and drives `internal/cli`'s functions
  against it - the check on `internal/testserver`'s own fidelity, run
  separately because it needs a Docker daemon and network access neither
  `go test ./...` nor CI's fast `test` job should require.
