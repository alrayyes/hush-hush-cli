# hush-hush-cli

[![CI](https://github.com/alrayyes/hush-hush-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/alrayyes/hush-hush-cli/actions)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/alrayyes/hush-hush-cli.svg)](https://pkg.go.dev/github.com/alrayyes/hush-hush-cli)
[![licence](https://img.shields.io/badge/licence-GPL--3.0-blue)](LICENSE)

Client for the [hush-hush](https://github.com/alrayyes/hush-hush) secrets
object store — the writer's and every consumer's interface to it: inject a
secret, fetch and decrypt one, list what's stored, rotate a value, delete
an object.

## Requirements

- **Go 1.27 or newer** to build from source.
- **[age](https://github.com/FiloSottile/age)**, to generate the keypairs a
  writer and a consumer each need. Not a dependency of this CLI itself — it
  only ever handles already-sealed ciphertext, never a private key or
  plaintext value outside a single `inject`/`get` call.
- A running `hush-hush` server to talk to.

## Installation

See [INSTALL.md](INSTALL.md) - AUR, `.deb`/`.rpm`, Docker, Nix,
`go install`, or from source.

## Usage

A consumer needs an age keypair to receive a secret -
[`age-keygen`](https://github.com/FiloSottile/age) generates one:

```sh
age-keygen -o consumer.key
# Public key: age1...
```

Inject a secret, sealed to one or more recipients (a write token comes from
`hush-hush token issue`, run against the server itself - see
[hush-hush's README](https://github.com/alrayyes/hush-hush#start-the-server)
for how, including inside a container):

```sh
export HUSH_HUSH_SERVER=http://localhost:8080
export HUSH_HUSH_TOKEN=9f8e7d6c...

echo -n "hunter2" | hush-hush-cli inject mattermost_deploy_webhook \
  --recipients age1... --used-by homelab/example-app \
  --description "prod deploy webhook"
```

`--recipients` is optional if every `--used-by` consumer is already
registered with a public key in the server's own consumer directory: omit
it and `inject` resolves each consumer's registered key from there instead,
failing clearly if one has none registered. Passing `--recipients`
explicitly always wins over that resolution:

```sh
hush-hush-cli inject mattermost_deploy_webhook --used-by homelab/example-app
```

Fetch and decrypt it - only whoever holds a matching private key can.
`--identity` takes the bare key, so pull it out of `age-keygen`'s comment
header first:

```sh
hush-hush-cli get mattermost_deploy_webhook --identity "$(tail -1 consumer.key)"
```

List what's stored - each object's `slug`, `used_by`, and `description`,
never the value itself, which is why this needs a token the same as
`inject`/`update`/`delete` do, unlike `get`:

```sh
hush-hush-cli list
hush-hush-cli list --json | jq '.[].slug'
hush-hush-cli list --used-by homelab/example-app
```

Label secrets with `--tag` on `inject` (repeatable or comma-separated; 1 to 32
characters from `a-z 0-9 . _ / -`, at most 10, lower-cased by the server).
`list` shows them in a TAGS column, and when each secret was created and
last updated in CREATED and UPDATED (`--json` adds who did it, as
`created_by` and `updated_by`). `list --tag prod` keeps only objects carrying
that tag; repeat the flag or comma-separate and an object must carry every
one. It combines with `--used-by`. `update --tag` replaces a secret's tags,
`update --clear-tags` removes them all, and `update` with neither leaves
them alone:

```sh
hush-hush-cli inject db_password --used-by homelab/example-app --tag prod,db
hush-hush-cli update db_password --tag staging
hush-hush-cli update db_password --clear-tags
```

Check what a single object is recorded as being used by - no credential
required, unlike `list`, since this only ever discloses what a caller
already knows the slug of:

```sh
hush-hush-cli used-by mattermost_deploy_webhook
hush-hush-cli used-by mattermost_deploy_webhook --json
```

Rotate the value, then remove the object once nothing needs it any more:

```sh
echo -n "new-value" | hush-hush-cli update mattermost_deploy_webhook \
  --recipients age1...
hush-hush-cli delete mattermost_deploy_webhook
```

`inject` and `update` both read the new plaintext from stdin rather than a
flag or argument, so it never ends up in shell history or a process
listing.

`update --used-by a,b` replaces a secret's consumers and seals the new value
to their registered public keys, the way `inject` does, unless `--recipients`
is given. `update --clear-used-by` removes every consumer (and needs
`--recipients`), and `update` with neither leaves them alone:

```sh
echo -n "new-value" | hush-hush-cli update mattermost_deploy_webhook \
  --used-by homelab/vps-docker
```

`inject --keep-readable-copy` and `update --keep-readable-copy` also seal the
value to your own escrowed identity key, so you can decrypt what you wrote.
The server never adds it for you, and the flag fails, without writing
anything, if you have no escrowed key yet. It needs hush-hush v2.54.0 or
later.

Query the audit trail - who touched an object, when, and how. Filters
combine with AND; `--since`/`--until` take RFC3339 timestamps:

```sh
hush-hush-cli audit-log --object mattermost_deploy_webhook --since 2026-09-01T00:00:00Z
hush-hush-cli audit-log --actor tok_abc123 --format json | jq '.[].action'
```

No credential is required - reading the audit trail needs no write token,
unlike `list`. `--actor` restricts to entries authenticated by a specific
verified actor (a token ID, or the admin account's own actor ID);
`--caller` restricts to entries recorded with a given self-presented
`--caller` value instead, which - unlike `--actor` - is never verified.
`--limit N` caps how many entries print, regardless of how many pages it
takes to fetch them; with no `--limit`, every matching entry prints. There
is no `--follow` - this is a bounded query, run it again to see what's new.

Check whether the target server has an admin account bootstrapped yet -
useful for telling a server with no admin account yet apart from one
that's just unreachable. No credential is required, and a server with no
admin account is a normal, exit-0 result:

```sh
hush-hush-cli status
hush-hush-cli status --json | jq .bootstrapped
```

Mint a consumer read token, scoped to one consumer, for `get`'s
`--consumer-token`/`HUSH_HUSH_CONSUMER_TOKEN` - a write token is required,
the same as `inject`/`update`/`delete`, since issuing a credential that
grants read access is itself a write-path operation:

```sh
hush-hush-cli token create homelab/vps-docker --ttl 720h --description "ci reader"
hush-hush-cli token list
hush-hush-cli token rotate <id> --ttl 720h
hush-hush-cli token revoke <id>
hush-hush-cli token purge <id>
```

`create` and `rotate` print the raw value once - it's never recoverable
again afterwards, and `list` never includes it. `--ttl` is required on
both, a Go duration (`720h`, not `30d`). `revoke` invalidates a token
without deleting its record, so it still shows in `list`, whose
`STATUS` column says `active`, `expired` or `revoked`; `purge` removes
an already-revoked or already-expired token's record for good, and
refuses one that's still active.

Manage the consumer directory - the names `--used-by` refers to, with an
optional age public key `inject` seals to. A write token is required:

```sh
hush-hush-cli consumer list --query homelab
hush-hush-cli consumer add homelab/new-device
hush-hush-cli consumer update homelab/new-device --public-key age1...
hush-hush-cli consumer update homelab/old-name --name homelab/new-name
hush-hush-cli consumer delete homelab/new-device
```

`update` renames and/or registers a key; a rename onto an existing name
merges the two. `delete` strips the consumer from every secret's
`used_by` list but never deletes a secret.

## Configuration

Settings are read in this order, each layer overriding the one before it:
**flags > environment variables > config file > defaults**. None of them
are required - environment variables alone are enough for a CI job or a
container - but `init` sets up a config file the first time it matters:

```sh
hush-hush-cli init      # writes ~/.config/hush-hush-cli/config.yaml
```

(`$XDG_CONFIG_HOME` instead of `~/.config` if it's set.) On a terminal,
`init` prompts for each setting in turn - server, token, caller,
recipients, identity - showing the current value or default so a bare
Enter accepts it. For `token` and `identity`, whatever you type is never
written to the file as-is by default: you're offered a choice of storing
it in the OS keyring, a command that retrieves it (`pass show ...` and
similar), the literal value, or not persisting it at all. `--yes`, or
running `init` with no terminal attached (a script), skips every prompt
and writes an empty starter file instead - nothing to answer, nothing
written that wasn't already there. `--force` overwrites an existing file;
without it, `init` refuses rather than touching one that's already there.

Run any other command with no config file and no relevant environment
variable set, on a terminal, and it offers to run through that same setup
before continuing - no need to remember to run `init` first. Every other
command reads configuration only; none of them prompt, and a value still
missing once flags/environment/file/defaults are all checked fails
immediately, naming the flag, the environment variable, and `init` as the
way to fix it.

| Flag                       | Environment variable               | config key               | Meaning                                                                                                                                                    |
| -------------------------- | ---------------------------------- | ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--server`                 | `HUSH_HUSH_SERVER`                 | `server`                 | Server base URL. Default `http://localhost:8080`.                                                                                                          |
| `--token`                  | `HUSH_HUSH_TOKEN`                  | `token`                  | Bearer token, required for `inject`/`update`/`delete`; also authorizes `get`, taking priority over `--consumer-token` when both are set.                   |
| `--token-command`          | `HUSH_HUSH_TOKEN_COMMAND`          | `token_command`          | Command whose trimmed stdout is the token instead - wins over `--token` if both are set.                                                                   |
| `--consumer-token`         | `HUSH_HUSH_CONSUMER_TOKEN`         | `consumer_token`         | Read-only, consumer-scoped bearer token, `get` only - used as a fallback when `--token` isn't set. Minted with `hush-hush-cli token create`.               |
| `--consumer-token-command` | `HUSH_HUSH_CONSUMER_TOKEN_COMMAND` | `consumer_token_command` | Command whose trimmed stdout is the consumer token instead - wins over `--consumer-token` if both are set.                                                 |
| `--caller`                 | `HUSH_HUSH_CALLER`                 | `caller`                 | Self-presented identity recorded in the audit log. Optional.                                                                                               |
| `--recipients`             | `HUSH_HUSH_RECIPIENTS`             | `recipients`             | Comma-separated age recipients, for `inject`/`update`. Wins over `--used-by` resolution.                                                                   |
| `--identity`               | `HUSH_HUSH_IDENTITY`               | `identity`               | Comma-separated age private keys, for `get`.                                                                                                               |
| `--identity-command`       | `HUSH_HUSH_IDENTITY_COMMAND`       | `identity_command`       | Command whose trimmed stdout is the identity instead - wins over `--identity` if both are set.                                                             |
| `--used-by`                | -                                  | -                        | Consumers of the secret (repeatable or comma-separated), `inject` only. With no `--recipients`, each consumer's registered public key is resolved instead. |
| `--json`                   | -                                  | -                        | Print the raw JSON array instead of a table, `list` only.                                                                                                  |
| `--description`            | -                                  | -                        | Free-text label, fixed at creation, `inject` only.                                                                                                         |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the toolchain, the hooks, and how
a change gets reviewed and released.

## Licence

[GPL-3.0](LICENSE).
