# Tasks

## 1. Dependency check

- [ ] 1.1 Bump `go.mod`'s `github.com/alrayyes/hush-hush-go/v4` pin from
      its current `v4.1.0` to `v4.1.2` (or later), run `go mod tidy`, and
      confirm `go build ./...` still succeeds. No SDK code change is
      needed for this feature (`design.md`'s Context — `WithAPIKey`
      already sends its credential on every call, `GetObject` included)
      but the bump picks up the codegen regen and keeps this repo off a
      stale pin. Verify: `go build ./...` succeeds against the bumped
      version.

## 2. Config resolution

- [ ] 2.1 Add `consumer_token` / `consumer_token_command` to the config
      struct and flag/env/config-file parsing, following the exact
      pattern `token` / `token_command` already uses (same file(s) that
      define those). Verify: a unit test setting `consumer_token` via
      each of flag, env var (`HUSH_HUSH_CONSUMER_TOKEN`) and config file
      resolves to the expected value, and flag > env > config file
      precedence holds (mirrors this change's `cli-config` delta spec,
      "Configuration precedence").
- [ ] 2.2 Wire `consumer_token_command` through the same command-runner
      used for `token_command`, including the non-zero-exit failure path.
      Verify: a unit test covers command-wins-over-literal and a failing
      command surfaces a named error rather than an empty/literal
      fallback (delta spec's "Credential fields resolve via command or
      literal value").

## 3. Credential selection and error handling

- [ ] 3.1 Give `internal/cli.Get` (`get.go:13`) its own client-
      construction step instead of `Config.newClient()` (`cli.go:65`):
      build the SDK client with `c.Token` if non-empty, else
      `c.ConsumerToken`, else empty (unchanged today's no-credential
      behavior). Leave `newClient()` and every other command untouched -
      only `Get` ever reads `ConsumerToken`. Verify: a unit test against
      `internal/testserver`'s fake confirms `Get` sends the write token
      when both are set, the consumer token when only it is set, and no
      credential when neither is set (delta spec's "Consumer read token
      used as a fallback on object fetches").
- [ ] 3.2 Extend the unauthorized-response handling around
      `client.go`'s existing `ErrUnauthorized` case so a rejected `get`
      names which credential was missing or wrong, per this change's
      `cli-config` delta ("Actionable error on a rejected read"). Verify:
      unit tests cover all three cases - neither credential configured,
      a rejected write token, and a rejected consumer-token-only
      request - checking each error's text names the right flag(s).

## 4. Documentation

- [ ] 4.1 Add `--consumer-token` / `HUSH_HUSH_CONSUMER_TOKEN` /
      `consumer_token` and the matching `-command` row to README.md's
      existing settings table (README.md:159-164), and a short note on
      what a consumer read token is for, where it's issued (the
      `hush-hush` web UI, not this CLI — proposal.md's "Not in scope"),
      and that a configured write token always takes priority over it on
      `get`. Verify: table renders correctly and the README's documented
      flags match what task 2.1 actually implemented.
- [ ] 4.2 Update CONTRIBUTING.md if it documents the config surface
      (check for a section listing config fields, same as README's
      table) so it doesn't fall out of sync. Verify: `grep -n token
CONTRIBUTING.md` reviewed and updated if it lists fields individually;
      otherwise note it needs no change.
