# Design

## Context

`internal/client.New(baseURL, token)` builds the SDK client with a
single credential: `hushhush.NewClient(baseURL,
hushhush.WithAPIKey(token))` (`client.go:60`), and `internal/cli.Config`
has one `newClient()` helper (`cli.go:65`) that every command — `Get`
included — calls with `c.Token`. There is exactly one `Authorization`
header per request; the SDK doesn't send two.

An earlier version of this design assumed `hushhush.WithAPIKey` only
attached its credential to write operations (create/update/delete) and
that `GetObject` needed a new, separate SDK option to carry a
read-scoped credential at all. That was wrong, confirmed by
`hush-hush-go` owner Ryan Kes closing
[`alrayyes/hush-hush-go#180`](https://github.com/alrayyes/hush-hush-go/issues/180#issuecomment-5874108681)
without a code change:

> this client's WithAPIKey token is already attached to every request
> unconditionally (client.go's authEditor, applied the same way to
> GetObject as to CreateObject) - it already sends Authorization:
> Bearer on reads whenever a key is configured, and a consumer read
> token works identically since the server doesn't care which field
> name you called it. Verified: alrayyes/hush-hush-go@ef79b59 (post-
> codegen-regen, released as 4.1.2).

So `WithAPIKey` already sends whatever credential it's given on every
call, `GetObject` included, and the server doesn't distinguish a write
token from a consumer token by field name — only by what it's bound to
and what `used_by` says. There is no SDK-side gap: `hush-hush-go` v4.1.2
(`ef79b59`, already released) is sufficient as-is, and
`alrayyes/hush-hush-go#180` is closed (`stateReason: COMPLETED`).

That leaves a purely CLI-side decision. `client.New`/`newClient()` take
exactly one token and hand it to exactly one `WithAPIKey` call, so this
repo can't send a write token and a consumer token on the same request
— it has to pick one credential per `Get` call.

## Goals / Non-Goals

**Goals:**

- Resolve `consumer_token` alongside the existing write `token`, and
  have `Get` send it when there's no write token configured.
- Keep every other command (`inject`, `update`, `delete`, `list`,
  `audit-log`) exactly as they are today — they still require and send
  only the write `token`, since `consumerBearerAuth` never authorizes
  anything besides `getObject`.

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

## Decisions

**`Get` picks one credential — the write `token` if set, else
`consumer_token` — rather than trying to send both.** The SDK client
carries a single `apiKey`, so there's one `Authorization` header per
request regardless of how many credentials are configured. A write
token already authorizes every read (`bearerAuth` covers `getObject`
too), so it takes priority when present; `consumer_token` is the
fallback for a caller that holds no write token at all, which is the
scenario proposal.md's Why describes — a read-only consumer. This
happens in `internal/cli`, not `internal/client`: `Config.newClient()`
(`cli.go:65`) is shared by every command and stays exactly as it is,
sending `c.Token`. `Get` (`get.go:13`) gets its own client-construction
step instead of calling the shared `newClient()`, choosing `c.Token` if
non-empty, else `c.ConsumerToken`, before calling `client.New`.

**No `hush-hush-go` dependency work.** The version already in `go.mod`
either already includes the `ef79b59`/v4.1.2 codegen regen or needs
nothing more than an ordinary `go get -u` bump to pick it up — check
`go.mod`'s current pinned version against v4.1.2 when implementing; if
it's already at or past that tag, task group 1 is just verifying the
version, not waiting on anything.

**`consumer_token` / `consumer_token_command` mirrors the existing
`token` / `token_command` shape exactly**, per the issue body's own
instruction to follow that precedent rather than invent a second
mechanism (`cli-config`'s "Credential fields resolve via command or
literal value" requirement, extended by this change's delta spec) — same
precedence order, same command-wins-over-literal rule, same
non-swallowed command-failure behavior.

## Risks / Trade-offs

- **A caller holding both a write token and a consumer token always
  reads with the write token, never exercising the consumer-scoped
  path.** That's fine for what this CLI needs — a write-token holder
  can already read anything — but it does mean there's no way to force
  a `get` to authenticate as a specific consumer for testing/debugging
  purposes without temporarily unsetting `token`. Not solving this now;
  worth an explicit flag later if it turns out to matter
  (`--consumer-token`-only mode), not blocking this change.
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

No data migration, and nothing external to wait on: (1) confirm
`go.mod`'s `hush-hush-go` pin is at or past v4.1.2 (bump if not), (2)
add the config fields and `Get`'s credential-selection step, (3)
README/CONTRIBUTING document it. Existing users who only use write
tokens or sessions are unaffected — `consumer_token` is an additive,
optional field that only ever changes behavior for a caller with no
write token configured.
