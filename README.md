# hush-hush-cli

[![CI](https://github.com/alrayyes/hush-hush-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/alrayyes/hush-hush-cli/actions)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/alrayyes/hush-hush-cli.svg)](https://pkg.go.dev/github.com/alrayyes/hush-hush-cli)
[![licence](https://img.shields.io/badge/licence-GPL--3.0-blue)](LICENSE)

Client for the [hush-hush](https://github.com/alrayyes/hush-hush) secrets
object store — the writer's and every consumer's interface to it: inject a
secret, fetch and decrypt one, rotate a value, delete an object.

## Status

The code has moved here from `hush-hush`, verified working, and has
shipped real releases with binaries, packages, and an AUR package - see
[INSTALL.md](INSTALL.md). `hush-hush` still ships its own copy of this
CLI for now; that copy is removed once packaging parity is confirmed -
[hush-hush-cli#6](https://github.com/alrayyes/hush-hush-cli/issues/6)
tracks what's left (a Docker image, Nix flake).

## Requirements

- **Go 1.27 or newer** to build from source.
- **[age](https://github.com/FiloSottile/age)**, to generate the keypairs a
  writer and a consumer each need. Not a dependency of this CLI itself — it
  only ever handles already-sealed ciphertext, never a private key or
  plaintext value outside a single `inject`/`get` call.
- A running `hush-hush` server to talk to.

## Installation

See [INSTALL.md](INSTALL.md) - AUR, `.deb`/`.rpm`, `go install`, or from
source.

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
container - but `init` writes a starter file the first time it matters:

```sh
hush-hush-cli init      # writes ~/.config/hush-hush-cli/config.yaml
```

(`$XDG_CONFIG_HOME` instead of `~/.config` if it's set.) Run the command
with nothing configured yet - no config file, no relevant environment
variable - on an interactive terminal, and it offers to write that starter
file itself before continuing; `--yes` skips the prompt and writes it
unconditionally, for a script that wants the file without a person to
answer for it. `--force` on `init` itself overwrites an existing file,
which the prompt never does.

| Flag              | Environment variable      | config key      | Meaning                                                                                  |
| ----------------- | ------------------------- | --------------- | ---------------------------------------------------------------------------------------- |
| `--server`        | `HUSH_HUSH_SERVER`        | `server`        | Server base URL. Default `http://localhost:8080`.                                        |
| `--token`         | `HUSH_HUSH_TOKEN`         | `token`         | Bearer token, for `inject`/`update`/`delete`.                                            |
| `--token-command` | `HUSH_HUSH_TOKEN_COMMAND` | `token_command` | Command whose trimmed stdout is the token instead - wins over `--token` if both are set. |
| `--caller`        | `HUSH_HUSH_CALLER`        | `caller`        | Self-presented identity recorded in the audit log. Optional.                             |
| `--recipients`    | `HUSH_HUSH_RECIPIENTS`    | `recipients`    | Comma-separated age recipients, for `inject`/`update`.                                   |
| `--identity`      | `HUSH_HUSH_IDENTITY`      | `identity`      | Comma-separated age private keys, for `get`.                                             |
| `--used-by`       | -                         | -               | Consumers of the secret (repeatable or comma-separated), `inject` only.                  |
| `--description`   | -                         | -               | Free-text label, fixed at creation, `inject` only.                                       |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the toolchain, the hooks, and how
a change gets reviewed and released.

## Licence

[GPL-3.0](LICENSE).
