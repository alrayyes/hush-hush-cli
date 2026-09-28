# Spec Delta

## MODIFIED Requirements

### Requirement: Configuration precedence

The CLI SHALL resolve each connection setting (server, token,
consumer_token, caller, recipients, identity, and each field's
`_command` sibling where one exists) from, in order of precedence:
command-line flag, then environment variable, then config file, then a
built-in default where one exists.

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
`token_command`, `consumer_token` / `consumer_token_command`, `identity`
/ `identity_command`), the CLI SHALL run the `_command` value through the
shell and use its trimmed output as the credential when both the literal
value and the command are set.

#### Scenario: Command wins over a literal value

- **WHEN** both `token` and `token_command` (or both `consumer_token` and
  `consumer_token_command`, or both `identity` and `identity_command`)
  are set, from any combination of flag, environment variable or config
  file
- **THEN** the CLI runs the command and uses its trimmed stdout as the
  credential, ignoring the literal value

#### Scenario: Command failure is reported, not swallowed

- **WHEN** a configured `_command` exits non-zero
- **THEN** the CLI fails the invocation with an error naming the command
  that failed, and does not fall back to an empty or literal credential

## ADDED Requirements

### Requirement: Consumer read token used as a fallback on object fetches

`get` SHALL send the resolved write `token` as its credential when one
is configured, and only fall back to a resolved `consumer_token` when no
write `token` is resolved. A request carries exactly one
`Authorization: Bearer` credential; `consumer_token` never overrides or
accompanies an already-configured write `token`.

#### Scenario: Write token takes priority

- **WHEN** `get` runs with both `token` and `consumer_token` resolved
  from any source
- **THEN** the request to fetch the object includes
  `Authorization: Bearer <token>`, not the consumer token

#### Scenario: Consumer token used when no write token is configured

- **WHEN** `get` runs with `consumer_token` resolved and no `token`
  resolved from any source
- **THEN** the request to fetch the object includes
  `Authorization: Bearer <consumer_token>`

#### Scenario: Neither credential configured

- **WHEN** `get` runs with neither `token` nor `consumer_token` resolved
  from any source
- **THEN** the CLI still attempts the fetch with no credential, the same
  as it does today

### Requirement: Actionable error on a rejected read

When a `GET /objects/{slug}` request is rejected with an unauthorized
response, the CLI SHALL fail with an error naming the credential that
was missing or invalid, rather than surfacing the raw HTTP failure.

#### Scenario: Unauthorized with neither credential configured

- **WHEN** `get` runs with neither `token` nor `consumer_token` resolved
  and the server responds unauthorized
- **THEN** the CLI fails with an error naming `--token`/`--consumer-token`
  and their environment variable and config-file forms as ways to
  provide one, not a raw HTTP status

#### Scenario: Unauthorized with a write token configured

- **WHEN** `get` runs with `token` resolved (whether or not
  `consumer_token` is also resolved) and the server responds
  unauthorized
- **THEN** the CLI fails with an error stating the configured write
  token was rejected, not a raw HTTP status - the same message a
  rejected `inject`/`update`/`delete` already gives

#### Scenario: Unauthorized with only a consumer token configured

- **WHEN** `get` runs with `consumer_token` resolved and no `token`
  resolved, and the server responds unauthorized
- **THEN** the CLI fails with an error stating the configured consumer
  token was rejected, not a raw HTTP status
