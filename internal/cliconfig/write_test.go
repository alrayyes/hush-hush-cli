package cliconfig_test

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/cliconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path) //nolint:gosec // path is built from t.TempDir(), not user input
	require.NoError(t, err)

	return string(content)
}

func TestWriteStarterWritesTheTemplateOwnerOnly(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")

	require.NoError(t, cliconfig.WriteStarter(path))

	assert.Equal(t, cliconfig.StarterConfig, readFile(t, path))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestWriteInteractiveWritesEveryPlainFieldAndLeavesSkippedCredentialsBlank(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	sc := bufio.NewScanner(strings.NewReader("https://example.com\ncaller-id\nage1recipient\n"))
	readPassword := func(int) ([]byte, error) { return []byte(""), nil }

	require.NoError(t, cliconfig.WriteInteractive(path, sc, new(bytes.Buffer), 0, readPassword, cliconfig.Values{}))

	content := readFile(t, path)
	assert.Contains(t, content, `server: "https://example.com"`)
	assert.Contains(t, content, `caller: "caller-id"`)
	assert.Contains(t, content, `recipients: "age1recipient"`)
	assert.Contains(t, content, `token: ""`)
	assert.Contains(t, content, `identity: ""`)
	// Left as the commented example, not written as a real key.
	assert.Contains(t, content, `# token_command: "pass show hush-hush/write-token"`)
	assert.NotContains(t, content, "\ntoken_command: \"")
	assert.NotContains(t, content, "\nidentity_command: \"")
}

func TestWriteInteractiveKeepsCurrentValuesOnABareEnter(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	sc := bufio.NewScanner(strings.NewReader("\n\n\n"))
	readPassword := func(int) ([]byte, error) { return []byte(""), nil }
	current := cliconfig.Values{Server: "http://kept:1", Caller: "me", Recipients: "age1kept"}

	require.NoError(t, cliconfig.WriteInteractive(path, sc, new(bytes.Buffer), 0, readPassword, current))

	content := readFile(t, path)
	assert.Contains(t, content, `server: "http://kept:1"`)
	assert.Contains(t, content, `caller: "me"`)
	assert.Contains(t, content, `recipients: "age1kept"`)
}

// Not parallel: keyring.MockInit swaps a package-level provider.
func TestWriteInteractiveKeyringChoiceStoresInTheKeyring(t *testing.T) {
	keyring.MockInit()

	path := filepath.Join(t.TempDir(), "config.yaml")
	// server, token-persistence(keyring), caller, recipients - identity
	// left blank, so it never reaches a persistence prompt.
	sc := bufio.NewScanner(strings.NewReader("\n1\n\n\n"))
	readPassword := func(int) ([]byte, error) { return []byte("s3cret-token"), nil }

	require.NoError(t, cliconfig.WriteInteractive(path, sc, new(bytes.Buffer), 0, readPassword, cliconfig.Values{}))

	content := readFile(t, path)
	assert.Contains(t, content, `token_command: "hush-hush-cli config keyring-get token"`)
	assert.NotContains(t, content, "s3cret-token")

	stored, err := keyring.Get(cliconfig.KeyringService, "token")
	require.NoError(t, err)
	assert.Equal(t, "s3cret-token", stored)
}

func TestWriteInteractiveCommandChoiceWritesTheGivenCommand(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	// server, caller, recipients blank; identity-persistence(command) plus
	// the command itself.
	sc := bufio.NewScanner(strings.NewReader("\n\n\n2\npass show hush-hush/identity-key\n"))

	call := 0
	readPassword := func(int) ([]byte, error) {
		call++
		if call == 1 {
			return []byte(""), nil // token: blank, skipped
		}

		return []byte("priv-key-value"), nil // identity
	}

	require.NoError(t, cliconfig.WriteInteractive(path, sc, new(bytes.Buffer), 0, readPassword, cliconfig.Values{}))

	content := readFile(t, path)
	assert.Contains(t, content, `identity_command: "pass show hush-hush/identity-key"`)
	assert.NotContains(t, content, "priv-key-value")
}

func TestWriteInteractiveLiteralChoiceWritesTheValueInTheClear(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	// server, token-persistence(literal), caller, recipients; identity blank.
	sc := bufio.NewScanner(strings.NewReader("\n3\n\n\n"))
	readPassword := func(int) ([]byte, error) { return []byte("a-literal-token"), nil }

	require.NoError(t, cliconfig.WriteInteractive(path, sc, new(bytes.Buffer), 0, readPassword, cliconfig.Values{}))

	content := readFile(t, path)
	assert.Contains(t, content, `token: "a-literal-token"`)
	assert.NotContains(t, content, "\ntoken_command: \"")
}
