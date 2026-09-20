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
  --recipients age1... --used-by homelab/vps-docker \
  --description "prod deploy webhook"
```

Fetch and decrypt it - only whoever holds a matching private key can.
`--identity` takes the bare key, so pull it out of `age-keygen`'s comment
header first:

```sh
hush-hush-cli get mattermost_deploy_webhook --identity "$(tail -1 consumer.key)"
```

List what's stored - each object's `id`, `used_by`, and `description`,
never the value itself, which is why this needs a token the same as
`inject`/`update`/`delete` do, unlike `get`:

```sh
hush-hush-cli list
hush-hush-cli list --json | jq '.[].id'
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
useful for telling an unbootstrapped server apart from one that's just
unreachable. No credential is required, and an unbootstrapped server is a
normal, exit-0 result:

```sh
hush-hush-cli status
hush-hush-cli status --json | jq .bootstrapped
```

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

| Flag                 | Environment variable         | config key         | Meaning                                                                                        |
| -------------------- | ---------------------------- | ------------------ | ---------------------------------------------------------------------------------------------- |
| `--server`           | `HUSH_HUSH_SERVER`           | `server`           | Server base URL. Default `http://localhost:8080`.                                              |
| `--token`            | `HUSH_HUSH_TOKEN`            | `token`            | Bearer token, for `inject`/`update`/`delete`.                                                  |
| `--token-command`    | `HUSH_HUSH_TOKEN_COMMAND`    | `token_command`    | Command whose trimmed stdout is the token instead - wins over `--token` if both are set.       |
| `--caller`           | `HUSH_HUSH_CALLER`           | `caller`           | Self-presented identity recorded in the audit log. Optional.                                   |
| `--recipients`       | `HUSH_HUSH_RECIPIENTS`       | `recipients`       | Comma-separated age recipients, for `inject`/`update`.                                         |
| `--identity`         | `HUSH_HUSH_IDENTITY`         | `identity`         | Comma-separated age private keys, for `get`.                                                   |
| `--identity-command` | `HUSH_HUSH_IDENTITY_COMMAND` | `identity_command` | Command whose trimmed stdout is the identity instead - wins over `--identity` if both are set. |
| `--used-by`          | -                            | -                  | Consumers of the secret (repeatable or comma-separated), `inject` only.                        |
| `--json`             | -                            | -                  | Print the raw JSON array instead of a table, `list` only.                                      |
| `--description`      | -                            | -                  | Free-text label, fixed at creation, `inject` only.                                             |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the toolchain, the hooks, and how
a change gets reviewed and released.

## Licence

[GPL-3.0](LICENSE).
