# hush-hush-cli

[![CI](https://github.com/alrayyes/hush-hush-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/alrayyes/hush-hush-cli/actions)
[![licence](https://img.shields.io/badge/licence-GPL--3.0-blue)](LICENSE)

Client for the [hush-hush](https://github.com/alrayyes/hush-hush) secrets
object store — the writer's and every consumer's interface to it: inject a
secret, fetch and decrypt one, rotate a value, delete an object.

## Status

This repo is a bootstrap shell, not yet a working client. `hush-hush-cli`
is splitting out of the `hush-hush` repo, which still owns it today —
see [`openspec/changes/split-cli-into-own-repo/`](openspec/changes/split-cli-into-own-repo/)
for the plan and [hush-hush-cli#1](https://github.com/alrayyes/hush-hush-cli/issues/1)
for progress. Until that migration lands, keep using the CLI documented in
[`hush-hush`'s own README](https://github.com/alrayyes/hush-hush#use-the-cli).

## Requirements

- **Go 1.27 or newer** to build from source.
- **[age](https://github.com/FiloSottile/age)**, to generate the keypairs a
  writer and a consumer each need. Not a dependency of this CLI itself — it
  only ever handles already-sealed ciphertext, never a private key or
  plaintext value outside a single `inject`/`get` call.
- A running `hush-hush` server to talk to.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the toolchain, the hooks, and how
a change gets reviewed and released.

## Licence

[GPL-3.0](LICENSE).
