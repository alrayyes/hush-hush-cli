## 1. Shared secret resolution (`token_command`/`identity_command`)

- [x] 1.1 Write a failing test in `internal/cliconfig` for a new
      `ResolveSecret(literal, command string) (string, error)`: command wins
      when both are set, literal is returned when only it is set, and a
      non-zero command exits with a wrapped error naming the command.
      Verify: `go test ./internal/cliconfig/...` fails for the right reason
      (function doesn't exist yet).
- [x] 1.2 Move `runSecretCommand` from `cmd/hush-hush-cli/main.go` into
      `internal/cliconfig` (unexported helper) and implement `ResolveSecret`
      on top of it. Verify: the 1.1 test passes.
- [x] 1.3 Update `config()` in `main.go` to call `cliconfig.ResolveSecret`
      for `token`/`token_command` instead of its inlined logic. Verify:
      `go test ./cmd/hush-hush-cli/...` still passes unchanged (existing
      token_command tests are the regression check).

## 2. Interactive prompting

- [x] 2.1 Write a failing test for a new `cliconfig.PromptValue(in
io.Reader, out io.Writer, label, current string) (string, error)`:
      bare Enter returns `current`, other input returns the trimmed typed
      value. Verify: `go test ./internal/cliconfig/...` fails, then
      implement and it passes.
- [x] 2.2 Write a failing test for a new `cliconfig.PromptSecret` that reads
      masked input via `golang.org/x/term.ReadPassword` given an injected
      file descriptor/reader seam testable without a real TTY (match the
      existing test approach in `cliconfig_test.go` for `Confirm`). Verify:
      test fails, then implement and it passes.
- [x] 2.3 Write a failing test for a new `cliconfig.PromptPersistence(in
io.Reader, out io.Writer, field string) (choice
cliconfig.PersistChoice, extra string, err error)` covering all four
      choices (keyring / command / literal / skip) by simulated input.
      Verify: test fails, then implement and it passes.
- [x] 2.4 Write a failing test for a new `cliconfig.PromptConfig(in
io.Reader, out io.Writer, existing cliconfig.Values) (cliconfig.Values,
error)` that walks server, token, caller, recipients, identity in
      order, using 2.1-2.3, and returns the assembled values plus each
      credential field's chosen persistence. Verify: test fails, then
      implement and it passes.

## 3. Keyring-backed persistence

- [x] 3.1 Add `github.com/zalando/go-keyring` to `go.mod` at a pinned exact
      version and run `go mod tidy` to update `go.sum`. Verify: `go build
./...` succeeds.
- [x] 3.2 Write a failing test for a new hidden `hush-hush-cli config
keyring-get <field>` command that reads a value previously written
      with `keyring.Set("hush-hush-cli", field, ...)` and prints it to
      stdout (skip/guard the test where no keyring backend is available in
      the CI environment, matching how other environment-dependent tests in
      this repo are skipped - check `integration/` for the existing
      pattern). Verify: test fails, then implement `newConfigCmd`/
      `newKeyringGetCmd` (both `Hidden: true`) and it passes.
- [x] 3.3 Wire `PersistChoiceKeyring` in `PromptConfig`'s caller: on that
      choice, call `keyring.Set` directly (in-process) and have the
      config-rendering step (task 4.2) write
      `<field>_command: "hush-hush-cli config keyring-get <field>"`.
      Verify: an `internal/cliconfig` or `cmd/hush-hush-cli` test exercises
      this end-to-end against a fake/injected keyring backend if
      `go-keyring` exposes one for tests, otherwise verify via the rendered
      config content only (mock the persistence step, not the OS keyring
      itself, in unit tests) and note the manual/real-keyring check in the
      task-completion notes.

## 4. `init` becomes interactive

- [x] 4.1 Write a failing test in `cmd/hush-hush-cli` (extend
      `init_test.go`) asserting that `init` with a simulated TTY-like
      reader (matching however `init_test.go` already fakes interactivity,
      if it does - otherwise extend the command to accept an injectable
      "is interactive" check the same way `maybeOfferInit` already uses
      `term.IsTerminal`) and no `--yes` prompts for all five fields and
      writes a config reflecting the answers, including a credential
      field's `_command` when the persistence choice was "command". Verify:
      test fails, then implement by calling `cliconfig.PromptConfig` from
      `init`'s `RunE` and rendering the result to YAML (replacing the fixed
      `writeStarterConfig` call for the interactive path only), and the
      test passes.
- [x] 4.2 Write/extend a test asserting `init --yes` and `init` with a
      non-interactive reader both still produce byte-identical output to
      today's `starterConfig` template. Verify: `go test
./cmd/hush-hush-cli/...` passes, confirming no regression.
- [x] 4.3 Verify `--force` behavior is unchanged (existing
      `errConfigAlreadyExists` test in `init_test.go`/`config_test.go`
      still passes as-is) and that `init --force` on an existing config re-
      runs the same interactive-or-template flow as 4.1/4.2. Verify: add a
      test for `--force` plus interactive input overwriting an existing
      file.

## 5. `identity_command`

- [x] 5.1 Write a failing test in `cmd/hush-hush-cli` (`get_test.go`)
      asserting `get` resolves `identity` via `cliconfig.ResolveSecret` when
      `identity_command`/`HUSH_HUSH_IDENTITY_COMMAND` is set, command
      winning over a literal `identity`/`HUSH_HUSH_IDENTITY` the same way
      `token_command` already wins for `token`. Verify: test fails, then
      wire `identity_command` as a new persistent-ish flag/viper binding in
      `get.go` (matching how `identity` itself is already bound) and the
      test passes.
- [x] 5.2 Add `identity_command` to the `starterConfig` template's comment
      block, mirroring how `token_command` is documented there.

## 6. Pre-command nudge uses the same interactive flow

- [x] 6.1 Write a failing test for `maybeOfferInit` confirming that, when
      the user confirms the "no config file found" prompt, it now produces
      the same interactively-populated config `init` would (not a blank
      template) - extend the existing `maybeOfferInit`-adjacent tests in
      `cli_test.go`/`config_test.go`. Verify: test fails, then refactor
      `maybeOfferInit` to call the same helper `init`'s `RunE` uses (task
      4.1) instead of `writeStarterConfig`, and the test passes.
- [x] 6.2 Confirm declining the nudge, or no TTY, still leaves defaults in
      place with the existing stderr nudge message unchanged (regression
      check on existing tests).

## 7. Token validation and clearer missing-value errors

- [x] 7.1 Write failing tests for `inject`, `update`, `delete` each
      returning a new `errNoToken`-style sentinel (naming `--token`,
      `HUSH_HUSH_TOKEN`, and `hush-hush-cli init`) when no token resolves
      from any source, and making no request to the test server
      (`internal/testserver`) in that case. Verify: tests fail, then add a
      `RequireToken bool` field to `cli.Config` (per design.md), set it
      `true` in `inject`/`update`/`delete` before calling `Validate()`,
      check it there, and the tests pass. `get` must be unaffected -
      confirm its existing "no token required" behavior still passes.
- [x] 7.2 Update `errNoRecipients` (`inject.go`, `update.go`) and
      `errNoIdentity` (`get.go`) messages to also name `hush-hush-cli init`.
      Update/extend the existing tests asserting these error messages.

## 8. Documentation

- [x] 8.1 Update `README.md`'s Configuration section: describe interactive
      `init`, the credential persistence choices, and add the
      `identity_command` row to the flag/env/config table. Verify: the
      documented commands and table match the implemented behavior exactly
      (read them back against the code).

## 9. Full verification

- [x] 9.1 Run `go test ./...` and `go test -tags=integration
./integration/...` and confirm both pass.
- [x] 9.2 Run the repo's lint/format hooks (`lefthook run pre-commit` or
      the equivalent `golangci-lint run` / `prettier`/`markdownlint`
      invocations this repo uses) and confirm they pass on the changed
      files, including the updated `README.md`.
