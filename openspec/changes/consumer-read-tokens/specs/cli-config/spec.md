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

### Requirement: Consumer read token sent on object fetches

The CLI SHALL send a resolved `consumer_token` as an
`Authorization: Bearer <consumer_token>` header on every `GET
/objects/{slug}` request, in addition to (not replacing) whatever write
credential the request already carries.

#### Scenario: Consumer token present

- **WHEN** `get` runs with a `consumer_token` resolved from any source
- **THEN** the request to fetch the object includes
  `Authorization: Bearer <consumer_token>`

#### Scenario: No consumer token configured

- **WHEN** `get` runs with no `consumer_token` resolved from any source
- **THEN** the CLI still attempts the fetch using whatever write
  credential is configured, sending no consumer-token header

### Requirement: Actionable error on a rejected read

When a `GET /objects/{slug}` request is rejected with an unauthorized
response, the CLI SHALL fail with an error naming the credential that
was missing or invalid, rather than surfacing the raw HTTP failure.

#### Scenario: Unauthorized with no consumer token configured

- **WHEN** `get` runs with no `consumer_token` resolved and the server
  responds unauthorized
- **THEN** the CLI fails with an error naming `--consumer-token`,
  `HUSH_HUSH_CONSUMER_TOKEN`, and `consumer_token` in the config file as
  ways to provide it, not a raw HTTP status

#### Scenario: Unauthorized with a consumer token configured

- **WHEN** `get` runs with a `consumer_token` resolved from any source
  and the server responds unauthorized
- **THEN** the CLI fails with an error stating the configured consumer
  token was rejected, not a raw HTTP status
