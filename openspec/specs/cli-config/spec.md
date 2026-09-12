## Purpose

Defines how `hush-hush-cli` resolves its connection settings (server,
token, caller, recipients, identity) across flags, environment variables
and a config file, and how it helps a user populate that config file
without ever putting a secret on the command line or in shell history.

## Requirements

### Requirement: Configuration precedence

The CLI SHALL resolve each connection setting (server, token, caller,
recipients, identity, and each field's `_command` sibling where one
exists) from, in order of precedence: command-line flag, then
environment variable, then config file, then a built-in default where one
exists.

#### Scenario: Flag overrides everything else

- **WHEN** a setting is present as a flag, an environment variable and in
  the config file simultaneously
- **THEN** the CLI uses the flag's value

#### Scenario: Environment variable overrides the config file

- **WHEN** a setting is present as an environment variable and in the
  config file, with no flag given
- **THEN** the CLI uses the environment variable's value

#### Scenario: Config file used when nothing else is set

- **WHEN** a setting is present only in the config file
- **THEN** the CLI uses the config file's value

### Requirement: Credential fields resolve via command or literal value

For each credential field that has a `_command` sibling (`token` /
`token_command`, `identity` / `identity_command`), the CLI SHALL run the
`_command` value through the shell and use its trimmed output as the
credential when both the literal value and the command are set.

#### Scenario: Command wins over a literal value

- **WHEN** both `token` and `token_command` (or both `identity` and
  `identity_command`) are set, from any combination of flag, environment
  variable or config file
- **THEN** the CLI runs the command and uses its trimmed stdout as the
  credential, ignoring the literal value

#### Scenario: Command failure is reported, not swallowed

- **WHEN** a configured `_command` exits non-zero
- **THEN** the CLI fails the invocation with an error naming the command
  that failed, and does not fall back to an empty or literal credential

### Requirement: Interactive init on a terminal

Running `init` with no `--yes` flag and standard input attached to a
terminal SHALL prompt the user, in turn, for the server URL, token,
caller identity, recipients and identity, showing each field's current
value (from an existing config file) or built-in default, before writing
the config file.

#### Scenario: Fresh interactive init

- **WHEN** `init` runs with no existing config file, `--yes` not given,
  and standard input is a terminal
- **THEN** the CLI prompts for server, token, caller, recipients and
  identity in turn, and writes a config file reflecting the answers

#### Scenario: Accepting the shown default

- **WHEN** a prompted field is left blank (bare Enter)
- **THEN** the CLI uses the value shown as that field's default and moves
  to the next field

### Requirement: Init falls back to a non-interactive template

Running `init` with `--yes`, or with standard input not attached to a
terminal, SHALL write the config file with no prompting.

#### Scenario: --yes skips every prompt

- **WHEN** `init --yes` runs, regardless of whether stdin is a terminal
- **THEN** the CLI writes the config file with no prompt, using built-in
  defaults for every field

#### Scenario: No terminal skips every prompt

- **WHEN** `init` runs with standard input not attached to a terminal and
  `--yes` was not given
- **THEN** the CLI writes the config file with no prompt, using built-in
  defaults for every field

### Requirement: Credential persistence choice

After an interactive `init` prompt collects a value for a credential
field (`token` or `identity`), the CLI SHALL offer the user a choice of
how to persist it: store it in the operating system's credential store, a
retrieval command the user supplies, the literal value written into the
config file, or not persisting it at all.

#### Scenario: Store in the OS keyring

- **WHEN** the user chooses the keyring option for a credential field
- **THEN** the CLI stores the value in the OS credential store and writes
  that field's `_command` sibling in the config file, pointing at a
  command that retrieves it from there

#### Scenario: User-supplied retrieval command

- **WHEN** the user chooses to supply their own retrieval command for a
  credential field
- **THEN** the CLI writes that command as the field's `_command` sibling
  in the config file, and does not write the literal value

#### Scenario: Literal value

- **WHEN** the user chooses to store the literal value for a credential
  field
- **THEN** the CLI writes the value directly into the config file for
  that field, at the same file permissions the config file already uses

#### Scenario: Skip persisting

- **WHEN** the user declines to persist a credential field
- **THEN** the CLI does not write that field's value or `_command` to the
  config file

### Requirement: Existing config file requires --force to overwrite

Running `init` when a config file already exists SHALL fail unless
`--force` is given, whether or not the run would otherwise be
interactive.

#### Scenario: init refuses without --force

- **WHEN** `init` runs and a config file already exists at the resolved
  path, without `--force`
- **THEN** the CLI fails with an error naming the existing path and
  `--force`, and does not modify the file

#### Scenario: init --force re-runs the same flow

- **WHEN** `init --force` runs and a config file already exists
- **THEN** the CLI overwrites it, following the same interactive-or-template
  behavior as a fresh `init` (Interactive init on a terminal / Init falls
  back to a non-interactive template)

### Requirement: First-run nudge offers the same interactive setup

Running any command other than `init` or `man` with no config file and no
relevant environment variable set, on a terminal, SHALL offer to run the
same interactive setup `init` itself provides, rather than writing an
unpopulated config file.

#### Scenario: Confirming the nudge runs interactive setup

- **WHEN** a command other than `init`/`man` is run with no config file,
  no relevant environment variable set, and standard input is a terminal,
  and the user confirms the offered setup
- **THEN** the CLI prompts for each field the same way `init` does, then
  continues the original command with the resulting configuration

#### Scenario: Declining or no terminal leaves defaults in place

- **WHEN** the user declines the nudge, or standard input is not a
  terminal
- **THEN** the CLI proceeds using built-in defaults with no config file
  written, and states on stderr that no config file or relevant
  environment variable was found

### Requirement: Commands never prompt for a missing value

Every command other than `init` SHALL resolve its configuration purely
from flags, environment variables, the config file and defaults, with no
interactive prompting of its own. A required value that is still missing
after that resolution SHALL fail the command with an error naming the
flag, the environment variable, and `init` as ways to provide it.

#### Scenario: Missing token on a write-path command

- **WHEN** `inject`, `update` or `delete` runs with no token resolved
  from any source
- **THEN** the command fails with an error naming `--token`,
  `HUSH_HUSH_TOKEN`, and `init`, and makes no request to the server

#### Scenario: Missing recipients on inject or update

- **WHEN** `inject` or `update` runs with no recipients resolved from any
  source
- **THEN** the command fails with an error naming `--recipients`,
  `HUSH_HUSH_RECIPIENTS`, and `init`

#### Scenario: Missing identity on get

- **WHEN** `get` runs with no identity resolved from any source
- **THEN** the command fails with an error naming `--identity`,
  `HUSH_HUSH_IDENTITY`, and `init`
