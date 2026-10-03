package cliconfig_test

import (
	"os"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/cliconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestRenderQuotesEveryValue(t *testing.T) {
	t.Parallel()

	got := cliconfig.Render("http://x:8080", cliconfig.RenderedSecret{Literal: `tok"en`}, "ci", "age1abc", cliconfig.RenderedSecret{Literal: "AGE-SECRET-KEY-1"})

	assert.Contains(t, got, `server: "http://x:8080"`)
	assert.Contains(t, got, `token: "tok\"en"`)
	assert.Contains(t, got, `caller: "ci"`)
	assert.Contains(t, got, `recipients: "age1abc"`)
	assert.Contains(t, got, `identity: "AGE-SECRET-KEY-1"`)
}

func TestRenderWritesACommandLineOnlyWhenOneWasChosen(t *testing.T) {
	t.Parallel()

	withCommand := cliconfig.Render("s", cliconfig.RenderedSecret{Command: "pass show t"}, "", "", cliconfig.RenderedSecret{Command: "pass show i"})
	assert.Contains(t, withCommand, "\ntoken_command: \"pass show t\"\n")
	assert.Contains(t, withCommand, "\nidentity_command: \"pass show i\"\n")

	without := cliconfig.Render("s", cliconfig.RenderedSecret{}, "", "", cliconfig.RenderedSecret{})
	assert.Contains(t, without, "\n# token_command: ", "the example stays commented out")
	assert.Contains(t, without, "\n# identity_command: ")
	assert.NotContains(t, without, "\ntoken_command:")
}

func TestPersistCredentialLiteralCommandAndSkip(t *testing.T) {
	t.Parallel()

	literal, err := cliconfig.PersistCredential(cliconfig.CredentialAnswer{Choice: cliconfig.PersistLiteral, Value: "v"}, "token")
	require.NoError(t, err)
	assert.Equal(t, cliconfig.RenderedSecret{Literal: "v"}, literal)

	command, err := cliconfig.PersistCredential(cliconfig.CredentialAnswer{Choice: cliconfig.PersistCommand, Extra: "pass show x"}, "token")
	require.NoError(t, err)
	assert.Equal(t, cliconfig.RenderedSecret{Command: "pass show x"}, command)

	skipped, err := cliconfig.PersistCredential(cliconfig.CredentialAnswer{Choice: cliconfig.PersistSkip}, "token")
	require.NoError(t, err)
	assert.Equal(t, cliconfig.RenderedSecret{}, skipped)
}

// Not parallel: keyring.MockInit swaps a package-level provider.
func TestPersistCredentialKeyringStoresTheValueAndWritesTheGetCommand(t *testing.T) {
	keyring.MockInit()

	got, err := cliconfig.PersistCredential(cliconfig.CredentialAnswer{Choice: cliconfig.PersistKeyring, Value: "s3cret"}, "identity")
	require.NoError(t, err)
	assert.Equal(t, cliconfig.RenderedSecret{Command: "hush-hush-cli config keyring-get identity"}, got)

	stored, err := keyring.Get(cliconfig.KeyringService, "identity")
	require.NoError(t, err)
	assert.Equal(t, "s3cret", stored)
}

// Not parallel: t.Setenv.
func TestAnyEnvVarSet(t *testing.T) {
	for _, name := range cliconfig.EnvVars {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}

	assert.False(t, cliconfig.AnyEnvVarSet())

	t.Setenv("HUSH_HUSH_CALLER", "")
	assert.True(t, cliconfig.AnyEnvVarSet(), "set but empty still counts as configured")
}

func TestStarterConfigNamesEveryField(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"server:", "token:", "token_command", "consumer_token:", "caller:", "recipients:", "identity:", "identity_command"} {
		assert.Contains(t, cliconfig.StarterConfig, field)
	}
}
