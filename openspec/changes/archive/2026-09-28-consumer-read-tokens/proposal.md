# Proposal

## Why

`GET /objects/{slug}` on `hush-hush` no longer accepts anonymous reads —
it now requires a write bearer token, an admin session, or a consumer
read token scoped to the object's `used_by` list
(`alrayyes/hush-hush#446`, merged to `main` as
`feat(api): consumer-scoped read tokens gating GET /objects/{slug}`, and
documented in `api/openapi.yaml`'s `consumerBearerAuth` security scheme).
A consumer that only ever reads — the common case `used_by` exists to
describe — has no write token to reach for, so this CLI needs its own way
to hold and send a consumer-scoped read token or every such consumer's
fetches start failing. Tracked as `alrayyes/hush-hush-cli#133`.

## What Changes

- Add a `consumer_token` / `consumer_token_command` config field pair,
  resolved with the same flag > env > config file > built-in default
  precedence and command-wins-over-literal behavior the existing `token`
  / `token_command` pair already has (`cli-config`'s "Configuration
  precedence" and "Credential fields resolve via command or literal
  value" requirements).
- Have `get` fall back to sending `consumer_token` as
  `Authorization: Bearer <consumer_token>` when no write `token` is
  configured — a write token already authorizes reads too, so it keeps
  taking priority; a consumer token authorizes reads only, per
  `consumerBearerAuth`'s scope.
- Report a 401 on a read with an actionable error naming the missing or
  invalid consumer token, rather than surfacing the raw HTTP failure —
  mirroring `internal/client.ErrUnauthorized`'s existing pattern for the
  write token.
- Document the new config fields in `README.md` and `CONTRIBUTING.md`.
- **Not in scope**: issuing, listing, rotating or revoking consumer
  tokens. `POST /consumer-tokens`, `GET /consumer-tokens`,
  `DELETE /consumer-tokens/{id}` and `POST /consumer-tokens/{id}/rotate`
  are all gated by `cookieAuth` (session) only in `api/openapi.yaml` — no
  bearer option — so token issuance is a `hush-hush` web-UI/admin-session
  operation. This CLI only ever consumes a token issued elsewhere and
  never creates one, so there's no new CLI subcommand for token
  lifecycle.

## Capabilities

### New Capabilities

(none — this extends how the CLI authenticates a read it already
performs, not a new capability)

### Modified Capabilities

- `cli-config`: adds the `consumer_token` / `consumer_token_command`
  field pair and their resolution/precedence behavior, following the
  same requirement shape as the existing `token` / `token_command` pair.

## Impact

- `internal/cli.Get`: picks which credential to hand the SDK client per
  call (write token if set, else consumer token) instead of using the
  shared `Config.newClient()` every other command uses unchanged. No
  `internal/client` or SDK change needed — `hushhush.WithAPIKey` already
  sends whatever credential it's given on every call, `GetObject`
  included (`alrayyes/hush-hush-go#180`, closed without a code change;
  see `design.md`).
- CLI config loading/flags: new `consumer_token` /
  `consumer_token_command` fields alongside the existing `token` /
  `token_command` ones.
- `README.md`, `CONTRIBUTING.md`: document the new configuration option.
- `go.mod`: bump the `github.com/alrayyes/hush-hush-go/v4` pin from
  `v4.1.0` to `v4.1.2` or later, a routine version bump rather than a
  blocking dependency. See `design.md`.
