# hush-hush-cli

[![CI](https://github.com/alrayyes/hush-hush-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/alrayyes/hush-hush-cli/actions)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/alrayyes/hush-hush-cli.svg)](https://pkg.go.dev/github.com/alrayyes/hush-hush-cli)
[![licence](https://img.shields.io/badge/licence-GPL--3.0-blue)](LICENSE)

Client for the [hush-hush](https://github.com/alrayyes/hush-hush) secrets
object store — the writer's and every consumer's interface to it: inject a
secret, fetch and decrypt one, rotate a value, delete an object.

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
`hush-hush token issue` on the server):

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

Rotate the value, then remove the object once nothing needs it any more:

```sh
echo -n "new-value" | hush-hush-cli update mattermost_deploy_webhook \
  --recipients age1...
hush-hush-cli delete mattermost_deploy_webhook
```

`inject` and `update` both read the new plaintext from stdin rather than a
flag or argument, so it never ends up in shell history or a process
listing.

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
| `--description`      | -                            | -                  | Free-text label, fixed at creation, `inject` only.                                             |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the toolchain, the hooks, and how
a change gets reviewed and released.

## Licence

[GPL-3.0](LICENSE).
