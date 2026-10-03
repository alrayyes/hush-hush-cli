package cliconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/zalando/go-keyring"
)

// KeyringService names hush-hush-cli's own OS keyring entries - one
// "account" per credential field ("token", "identity").
const KeyringService = "hush-hush-cli"

// EnvVars are every HUSH_HUSH_* variable a command reads - the root
// command's persistent flags plus recipients/identity, which are bound
// per-subcommand rather than on root. Used only to decide whether the
// tool is already configured through the environment, not to read a
// value.
var EnvVars = []string{
	"HUSH_HUSH_SERVER", "HUSH_HUSH_TOKEN", "HUSH_HUSH_TOKEN_COMMAND", "HUSH_HUSH_CALLER",
	"HUSH_HUSH_RECIPIENTS", "HUSH_HUSH_IDENTITY",
	"HUSH_HUSH_CONSUMER_TOKEN", "HUSH_HUSH_CONSUMER_TOKEN_COMMAND",
}

// StarterConfig is the blank template `init --yes` and a non-interactive
// init write.
const StarterConfig = `# hush-hush-cli config file. Flags and HUSH_HUSH_* environment variables
# both override these - see README.md#configuration.
server: http://localhost:8080
token: ""
# token_command runs a command and uses its trimmed stdout as the token
# instead - it wins over the literal value above if both are set.
# token_command: "pass show hush-hush/write-token"
# consumer_token is a read-only, consumer-scoped token - only used by
# get, and only when token above is empty. Mint one with
# "hush-hush-cli token create <consumer> --ttl <duration>".
consumer_token: ""
# consumer_token_command works the same as token_command, for consumer_token.
# consumer_token_command: "pass show hush-hush/consumer-token"
caller: ""
recipients: ""
identity: ""
# identity_command runs a command and uses its trimmed stdout as the
# identity instead - it wins over the literal value above if both are set.
# identity_command: "pass show hush-hush/identity-key"
`

// RenderedSecret is a credential field's config-file representation after
// its persistence choice has been applied: at most one of Literal/Command
// is non-empty (PersistSkip leaves both empty, matching an unanswered
// field).
type RenderedSecret struct {
	Literal string
	Command string
}

// PersistCredential turns one prompted credential answer into its
// config-file representation, storing the value in the OS keyring first
// when that's the chosen persistence.
func PersistCredential(answer CredentialAnswer, field string) (RenderedSecret, error) {
	switch answer.Choice {
	case PersistKeyring:
		if err := keyring.Set(KeyringService, field, answer.Value); err != nil {
			return RenderedSecret{}, fmt.Errorf("store %s in keyring: %w", field, err)
		}

		return RenderedSecret{Command: "hush-hush-cli config keyring-get " + field}, nil
	case PersistCommand:
		return RenderedSecret{Command: answer.Extra}, nil
	case PersistLiteral:
		return RenderedSecret{Literal: answer.Value}, nil
	default: // PersistSkip
		return RenderedSecret{}, nil
	}
}

// Render builds the YAML an interactive init writes. Every value is
// double-quoted via strconv.Quote regardless of content - simpler and
// safer than deciding case by case which values need it, at the cost of
// looking less like StarterConfig's own hand-written, selectively-quoted
// style; that constant is untouched and still what --yes/no-TTY writes.
func Render(server string, token RenderedSecret, caller, recipients string, identity RenderedSecret) string {
	var b strings.Builder

	b.WriteString("# hush-hush-cli config file. Flags and HUSH_HUSH_* environment variables\n")
	b.WriteString("# both override these - see README.md#configuration.\n")
	fmt.Fprintf(&b, "server: %s\n", strconv.Quote(server))
	fmt.Fprintf(&b, "token: %s\n", strconv.Quote(token.Literal))
	b.WriteString("# token_command runs a command and uses its trimmed stdout as the token\n")
	b.WriteString("# instead - it wins over the literal value above if both are set.\n")

	if token.Command != "" {
		fmt.Fprintf(&b, "token_command: %s\n", strconv.Quote(token.Command))
	} else {
		b.WriteString("# token_command: \"pass show hush-hush/write-token\"\n")
	}

	fmt.Fprintf(&b, "caller: %s\n", strconv.Quote(caller))
	fmt.Fprintf(&b, "recipients: %s\n", strconv.Quote(recipients))
	fmt.Fprintf(&b, "identity: %s\n", strconv.Quote(identity.Literal))
	b.WriteString("# identity_command runs a command and uses its trimmed stdout as the\n")
	b.WriteString("# identity instead - it wins over the literal value above if both are set.\n")

	if identity.Command != "" {
		fmt.Fprintf(&b, "identity_command: %s\n", strconv.Quote(identity.Command))
	} else {
		b.WriteString("# identity_command: \"pass show hush-hush/identity-key\"\n")
	}

	return b.String()
}

// AnyEnvVarSet reports whether any HUSH_HUSH_* variable a command reads is
// set, even to an empty value.
func AnyEnvVarSet() bool {
	for _, name := range EnvVars {
		if _, ok := os.LookupEnv(name); ok {
			return true
		}
	}

	return false
}
