## 1. Bootstrap this repo

- [x] 1.1 Create the GitHub repo, set PR merge settings, and verify branch protection is configured on `main`
- [x] 1.2 Wire CI (build/vet/test/lint/govulncheck/actionlint/markdown/commits/goreleaser-check/package) and verify a placeholder module passes it green
- [x] 1.3 Wire the release pipeline (release-please + goreleaser) and verify `goreleaser check` and a local snapshot build both succeed
- [x] 1.4 Add README/CONTRIBUTING/SECURITY/LICENSE/CLAUDE.md and verify they accurately describe the current bootstrap state
- [x] 1.5 Run `openspec init` and create this change

## 2. Migrate the code

- [x] 2.1 Move `cmd/hush-hush-cli`, `internal/cli`, `internal/client`, `internal/seal`, `internal/cliconfig` verbatim from `hush-hush` and verify `go build ./...` succeeds with import paths updated
- [x] 2.2 Resolve design.md's Open Question (Prism-mock vs. real-server-in-CI), replace the `newTestServer(t)` httptest-against-real-server helper accordingly, and verify every existing test still passes against the replacement
- [x] 2.3 Verify `golangci-lint run ./...` reports 0 issues on the moved code

## 3. Verify parity

- [x] 3.1 Build the real `hush-hush-cli` binary from this repo and verify it round-trips inject/get/update/delete against a running `hush-hush` server, matching `hush-hush`'s own README examples
- [x] 3.2 Verify config precedence (flags > environment > config file > defaults) and the `--token-command` feature both still work

## 4. First release and packaging (follow-up, doesn't block the migration itself)

- [ ] 4.1 Cut this repo's first real release and verify a binary is attached to it (tracked as [#13](https://github.com/alrayyes/hush-hush-cli/issues/13))
- [ ] 4.2 Refile `hush-hush#152` (Docker image), `hush-hush#139` (Nix flake CI verification), `hush-hush#153` (`flake.lock` update automation) as issues here, and resolve each (refiled as [#7](https://github.com/alrayyes/hush-hush-cli/issues/7)/[#8](https://github.com/alrayyes/hush-hush-cli/issues/8)/[#9](https://github.com/alrayyes/hush-hush-cli/issues/9); resolving each is still open)
- [ ] 4.3 Restore man-page generation (`cmd/hush-hush-cli/man.go`, the `go run ./cmd/hush-hush-cli man manpages` goreleaser hook, and the manpage entries in `.goreleaser.yml`'s `archives`/`nfpms`, per that file's own comment) and verify `man hush-hush-cli` works after a package install (tracked as [#14](https://github.com/alrayyes/hush-hush-cli/issues/14))

## 5. Remove the old copy from hush-hush

- [ ] 5.1 Once 3.1 and 3.2 are verified, remove `cmd/hush-hush-cli`, `internal/cli`, `internal/client`, `internal/seal`, `internal/cliconfig`, the CLI's AUR/deb/rpm/Nix packaging (PKGBUILD, `flake.nix`, the `.goreleaser.yml` `hush-hush-cli` build/archive/nfpm blocks), and its pact consumer test from `hush-hush`, and verify `hush-hush`'s own CI stays green with only the server left
- [ ] 5.2 Update `hush-hush`'s README to point here instead of documenting the CLI directly, and verify no stale CLI usage instructions remain there
