# Tasks

## 1. External prerequisite

- [x] 1.1 Track the SDK-side prerequisite: `alrayyes/hush-hush-go#180`
      already asks for consumer-token support (filed independently,
      found while researching this change — not new). It's unblocked
      now that `alrayyes/hush-hush#438` closed. Linked both ways:
      `alrayyes/hush-hush-cli#133` is now blocked-by
      `alrayyes/hush-hush-go#180`. Verify: `gh issue view 133 --repo alrayyes/hush-hush-cli` shows the blocked-by relationship.
- [ ] 1.2 Once `hush-hush-go#180` ships a release, bump this repo's
      `go.mod`/`go.sum` to it and run `go build ./...` to confirm the new
      option compiles against what this repo calls. Verify: `go build`
      succeeds with the bumped dependency.

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

## 3. Sending the token and handling rejection

- [ ] 3.1 In `internal/client`, construct the SDK client with
      `hushhush.WithConsumerToken` (from the bumped SDK, task 1.2) when a
      `consumer_token` resolves, alongside the existing `WithAPIKey` call
      in `client.go`. Verify: a test against `internal/testserver`'s fake
      confirms `GetObject` sends the consumer-token header when
      configured, and omits it when not (delta spec's "Consumer read
      token sent on object fetches").
- [ ] 3.2 Extend the unauthorized-response handling around
      `client.go`'s existing `ErrUnauthorized` case so a rejected `get`
      names the missing/invalid consumer token specifically, per this
      change's `cli-config` delta ("Actionable error on a rejected
      read"). Verify: a unit test covers both the no-token-configured and
      wrong-token-configured cases and checks the error text names
      `--consumer-token` / `HUSH_HUSH_CONSUMER_TOKEN` / config file in
      the first case.

## 4. Documentation

- [ ] 4.1 Add `--consumer-token` / `HUSH_HUSH_CONSUMER_TOKEN` /
      `consumer_token` and the matching `-command` row to README.md's
      existing settings table (README.md:159-164), and a short note on
      what a consumer read token is for and where it's issued (the
      `hush-hush` web UI, not this CLI — proposal.md's "Not in scope").
      Verify: table renders correctly and the README's documented flags
      match what task 2.1 actually implemented.
- [ ] 4.2 Update CONTRIBUTING.md if it documents the config surface
      (check for a section listing config fields, same as README's
      table) so it doesn't fall out of sync. Verify: `grep -n token
CONTRIBUTING.md` reviewed and updated if it lists fields
      individually; otherwise note it needs no change.
