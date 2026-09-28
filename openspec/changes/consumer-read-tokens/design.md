# Design

## Context

`internal/client.New` builds the SDK client with a single credential:
`hushhush.NewClient(baseURL, hushhush.WithAPIKey(token))`
(`internal/client/client.go:60`), and `GetObject` (`client.go:91`) calls
`c.sdk.GetObject(ctx, id, c.Caller)` with no separate credential
parameter — `WithAPIKey`'s doc comment says outright it's "the bearer
credential used on write operations." `hush-hush-go` at `main`
(`efa54ce`, 2026-09-28 04:34 UTC, before `hush-hush#446` merged at 15:17
UTC the same day) has no notion of a second, read-scoped bearer token:
there's one `apiKey` field on `config`, and `GetObject`'s signature
carries no room for one. See proposal.md - Why for the server-side
contract this closes the gap with.

This repo's `CLAUDE.md` gotchas are explicit that the SDK is the only
sanctioned way to talk to `hush-hush` — `internal/client` exists to wrap
it "for exactly this reason," with the one carved-out exception being
`hush-hush`'s own integration test on the server side. That rules out
reaching for `net/http` directly in `internal/client` to route around the
gap.

## Goals / Non-Goals

**Goals:**

- Resolve and send a `consumer_token` distinct from the existing write
  `token`, through the SDK.
- Keep the write-token path (`WithAPIKey`, create/update/delete) exactly
  as it is today — consumer tokens are additive, not a replacement.

**Non-Goals:**

- Issuing, rotating, listing or revoking consumer tokens from this CLI
  (proposal.md's "Not in scope" — that's `cookieAuth`-only, a web-UI/
  admin-session operation).
- Prompting for `consumer_token` in the interactive `init` flow.
  `init` today prompts for the fields every user needs to do anything
  useful (server, token, caller, recipients, identity); a consumer
  token is only relevant to a subset of users acting purely as read-only
  consumers, so it follows `cli-config`'s general precedence rule
  (flag/env/config file/default) without an `init` prompt or a
  persistence-choice step. Revisit if usage shows people expect it in
  `init`.
- Changing `hush-hush-go`'s own public API surface within _this_
  change's scope. That SDK is a separate repo with its own release
  cycle; this change depends on a release of it, but doesn't modify it.

## Decisions

**Add a `WithConsumerToken` SDK option and thread it through
`GetObject`, rather than reaching for raw HTTP.** The alternative —
having `internal/client` send the `Authorization: Bearer` header itself
around the generated client, or bypass the SDK for this one call — was
considered and rejected: it's exactly the workaround this repo's
standing SDK-only rule exists to prevent, and it would leave
`hush-hush-go` never learning about a token type the wire contract
(`consumerBearerAuth` in `api/openapi.yaml`) already documents. The
`WithAPIKey`/`config.apiKey` naming and doc-comment pattern in
`hush-hush-go`'s `client.go` gives a template to follow: a parallel
`WithConsumerToken(token string) Option` setting a `consumerToken` field
on `config`, and `GetObject` (or a new call the generated client already
exposes once regenerated against the updated spec) sending it as
`Authorization: Bearer <consumerToken>` when set, leaving the existing
write-token behavior on every other endpoint untouched.

This is a change to `hush-hush-go`, not to this repo, so it's a
prerequisite this change depends on rather than a task this repo's
`tasks.md` can carry out itself. Filing that as its own tracked issue on
`hush-hush-go` (rather than leaving it as an implicit assumption here) is
part of this change's task list.

**`consumer_token` / `consumer_token_command` mirrors the existing
`token` / `token_command` shape exactly**, per the issue body's own
instruction to follow that precedent rather than invent a second
mechanism (`cli-config`'s "Credential fields resolve via command or
literal value" requirement, extended by this change's delta spec) — same
precedence order, same command-wins-over-literal rule, same
non-swallowed command-failure behavior.

**A consumer token is sent whenever resolved, with no flag to disable
it**, since `consumerBearerAuth` only ever adds access (an object whose
`used_by` includes the token's bound consumer) and never removes it — a
`get` that already succeeds via a write token or session keeps
succeeding with a consumer-token header attached, so there's no case
where sending it unconditionally changes a successful outcome to a
failing one.

## Risks / Trade-offs

- **This CLI change is blocked on an external release.** Nothing in
  `hush-hush-cli` can send a consumer token until `hush-hush-go` ships
  `WithConsumerToken` (or equivalent) and this repo bumps its
  `go.mod` to that version → mitigated by filing the SDK-side issue now
  (this change's tasks.md) rather than discovering the gap mid-`apply`,
  and by scoping this change's own tasks so everything except the final
  wiring can proceed (config fields, flag/env parsing, error message,
  docs) while that's pending.
- **A consumer token scoped to the wrong consumer name is
  indistinguishable, from the CLI's perspective, from no token at all** —
  both produce the same unauthorized response, per `api/openapi.yaml`'s
  note that an out-of-scope consumer token gets the same 404 an unknown
  slug would (not 401/403), specifically so it can't be used to enumerate
  slugs → the "Unauthorized with a consumer token configured" scenario in
  this change's `cli-config` delta only covers the literal-401 case; a
  wrong-scope token surfaces as an ordinary 404, which `get` already
  reports as "object not found." No special-casing needed, but worth
  documenting in the README so a misconfigured consumer name doesn't read
  as a fetch bug.

## Migration Plan

No data migration. Rollout is: (1) `hush-hush-go` ships consumer-token
support, (2) this repo bumps its dependency and implements the config
fields, header-sending and error handling, (3) README/CONTRIBUTING
document it. Existing users who only use write tokens or sessions are
unaffected — `consumer_token` is an additive, optional field.
