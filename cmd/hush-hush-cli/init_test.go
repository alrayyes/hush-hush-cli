package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestInitWritesAStarterConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"init"})
	require.NoError(t, root.Execute())

	content, err := os.ReadFile(filepath.Join(dir, "hush-hush-cli", "config.yaml")) //nolint:gosec // path is built from t.TempDir(), not user input
	require.NoError(t, err)
	require.Contains(t, string(content), "server: http://localhost:8080")
}

func TestInitRefusesToOverwriteAnExistingFileWithoutForce(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	path := filepath.Join(dir, "hush-hush-cli", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("server: https://edited\n"), 0o600))

	root := newRootCmd()
	root.SetArgs([]string{"init"})
	require.Error(t, root.Execute())

	content, err := os.ReadFile(path) //nolint:gosec // path is built from t.TempDir(), not user input
	require.NoError(t, err)
	require.Equal(t, "server: https://edited\n", string(content))
}

func TestInitForceOverwritesAnExistingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	path := filepath.Join(dir, "hush-hush-cli", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("server: https://edited\n"), 0o600))

	root := newRootCmd()
	root.SetArgs([]string{"init", "--force"})
	require.NoError(t, root.Execute())

	content, err := os.ReadFile(path) //nolint:gosec // path is built from t.TempDir(), not user input
	require.NoError(t, err)
	require.Contains(t, string(content), "server: http://localhost:8080")
}

// TestUnconfiguredNonInteractiveRunProceedsOnDefaults confirms the nudge
// never blocks a script or CI job on a prompt nothing will answer: no
// config file, no HUSH_HUSH_* environment variable, and go test's own
// stdin is never a TTY.
func TestUnconfiguredNonInteractiveRunProceedsOnDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	srv, _, _ := testserver.New(t)

	root := newRootCmd()
	root.SetArgs([]string{"get", "nope", "--server", srv.URL})
	var errOut bytes.Buffer
	root.SetErr(&errOut)

	err := root.Execute()
	require.Error(t, err) // unknown object - the command itself still ran
	require.Contains(t, errOut.String(), "no config file")

	require.NoFileExists(t, filepath.Join(dir, "hush-hush-cli", "config.yaml"))
}

// TestYesFlagWritesAStarterConfigWithNoPrompt is rules/cli.md's "an
// explicit -y/--yes... opts into generating the file without asking" -
// the one way to get the starter file written outside an interactive
// terminal.
func TestYesFlagWritesAStarterConfigWithNoPrompt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	srv, _, _ := testserver.New(t)

	root := newRootCmd()
	root.SetArgs([]string{"get", "nope", "--server", srv.URL, "--yes"})

	_ = root.Execute() // unknown object - irrelevant to this test

	require.FileExists(t, filepath.Join(dir, "hush-hush-cli", "config.yaml"))
}

// TestRunInteractiveInitWritesEveryPlainFieldAndLeavesSkippedCredentialsBlank
// exercises runInteractiveInit directly rather than through root.Execute():
// go test's own stdin is never a TTY (see the comment above), so there's
// no way to drive the interactive branch of init's own RunE end-to-end
// without faking a real terminal. Calling the flow it delegates to
// directly, with a fake readPassword, covers the same behavior without
// that.
func TestRunInteractiveInitWritesEveryPlainFieldAndLeavesSkippedCredentialsBlank(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	path := filepath.Join(dir, "hush-hush-cli", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))

	root := newRootCmd()
	input := strings.NewReader("https://example.com\ncaller-id\nage1recipient\n")
	var out bytes.Buffer
	root.SetOut(&out)

	readPassword := func(int) ([]byte, error) { return []byte(""), nil }

	require.NoError(t, runInteractiveInit(root, path, bufio.NewScanner(input), readPassword))

	content, err := os.ReadFile(path) //nolint:gosec // path is built from t.TempDir(), not user input
	require.NoError(t, err)

	assert.Contains(t, string(content), `server: "https://example.com"`)
	assert.Contains(t, string(content), `caller: "caller-id"`)
	assert.Contains(t, string(content), `recipients: "age1recipient"`)
	assert.Contains(t, string(content), `token: ""`)
	assert.Contains(t, string(content), `identity: ""`)
	// Left as the commented example, not written as a real key -
	// NotContains on the bare prefix would also match inside that comment.
	assert.Contains(t, string(content), `# token_command: "pass show hush-hush/write-token"`)
	assert.Contains(t, string(content), `# identity_command: "pass show hush-hush/identity-key"`)
	assert.NotContains(t, string(content), "\ntoken_command: \"")
	assert.NotContains(t, string(content), "\nidentity_command: \"")
}

// TestRunInteractiveInitKeyringChoiceStoresInTheKeyring doesn't run
// t.Parallel(): MockInit swaps the package-level keyring backend, which
// isn't safe to share with a concurrently-running test expecting the real
// one (see config_keyring_test.go).
func TestRunInteractiveInitKeyringChoiceStoresInTheKeyring(t *testing.T) {
	keyring.MockInit()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	path := filepath.Join(dir, "hush-hush-cli", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))

	root := newRootCmd()
	// server, token-persistence(keyring), caller, recipients - identity
	// left blank, so it never reaches a persistence prompt.
	input := strings.NewReader("\n1\n\n\n")
	root.SetOut(new(bytes.Buffer))

	readPassword := func(int) ([]byte, error) { return []byte("s3cret-token"), nil }

	require.NoError(t, runInteractiveInit(root, path, bufio.NewScanner(input), readPassword))

	content, err := os.ReadFile(path) //nolint:gosec // path is built from t.TempDir(), not user input
	require.NoError(t, err)
	assert.Contains(t, string(content), `token_command: "hush-hush-cli config keyring-get token"`)
	assert.NotContains(t, string(content), "s3cret-token")

	stored, err := keyring.Get(keyringService, "token")
	require.NoError(t, err)
	assert.Equal(t, "s3cret-token", stored)
}

func TestRunInteractiveInitCommandChoiceWritesTheGivenCommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	path := filepath.Join(dir, "hush-hush-cli", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))

	root := newRootCmd()
	// server, caller, recipients blank; identity-persistence(command) plus
	// the command itself.
	input := strings.NewReader("\n\n\n2\npass show hush-hush/identity-key\n")
	root.SetOut(new(bytes.Buffer))

	call := 0
	readPassword := func(int) ([]byte, error) {
		call++
		if call == 1 {
			return []byte(""), nil // token: blank, skipped
		}

		return []byte("priv-key-value"), nil // identity
	}

	require.NoError(t, runInteractiveInit(root, path, bufio.NewScanner(input), readPassword))

	content, err := os.ReadFile(path) //nolint:gosec // path is built from t.TempDir(), not user input
	require.NoError(t, err)
	assert.Contains(t, string(content), `identity_command: "pass show hush-hush/identity-key"`)
	assert.NotContains(t, string(content), "priv-key-value")
}

func TestRunInteractiveInitLiteralChoiceWritesTheValueInTheClear(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	path := filepath.Join(dir, "hush-hush-cli", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))

	root := newRootCmd()
	// server, token-persistence(literal), caller, recipients; identity blank.
	input := strings.NewReader("\n3\n\n\n")
	root.SetOut(new(bytes.Buffer))

	readPassword := func(int) ([]byte, error) { return []byte("a-literal-token"), nil }

	require.NoError(t, runInteractiveInit(root, path, bufio.NewScanner(input), readPassword))

	content, err := os.ReadFile(path) //nolint:gosec // path is built from t.TempDir(), not user input
	require.NoError(t, err)
	assert.Contains(t, string(content), `token: "a-literal-token"`)
	assert.NotContains(t, string(content), "\ntoken_command: \"")
}

// TestConfigNudgeNeverBlocksARunTheEnvironmentAlreadyConfigures is a
// regression test: an XDG_CONFIG_HOME under a directory this process
// can't write into must not stop a command from running when
// HUSH_HUSH_SERVER already configures everything it needs - the CI job
// this tool's own spec commits to running "unmodified inside" is exactly
// the case that sets environment variables and nothing else, and may not
// have a writable home directory to resolve one against at all.
func TestConfigNudgeNeverBlocksARunTheEnvironmentAlreadyConfigures(t *testing.T) {
	unwritable := t.TempDir()
	require.NoError(t, os.Chmod(unwritable, 0o500))                       //nolint:gosec // G302 checks file perms; this chmod is on a directory
	t.Cleanup(func() { require.NoError(t, os.Chmod(unwritable, 0o700)) }) //nolint:gosec // G302 checks file perms; this chmod is on a directory - TempDir cleanup also needs write back

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(unwritable, "config"))
	viper.Reset()

	srv, _, _ := testserver.New(t)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)

	root := newRootCmd()
	root.SetArgs([]string{"get", "nope"})

	err := root.Execute()
	require.Error(t, err) // unknown object - the command itself still ran
}
