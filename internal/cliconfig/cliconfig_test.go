package cliconfig_test

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/cliconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errNoTTY is the fake PasswordReader failure TestPromptSecretPropagatesAReadFailure
// injects - a wrapped static error rather than a fresh errors.New at the
// call site (err113).
var errNoTTY = errors.New("no tty")

// scan is a small test helper: most of the Prompt* API takes a
// *bufio.Scanner (rather than a bare io.Reader) precisely so a caller can
// share one across several sequential prompts - see Confirm's own doc
// comment. Most tests here only ever make one read, so this just saves
// repeating bufio.NewScanner(strings.NewReader(...)) at every call site.
func scan(s string) *bufio.Scanner {
	return bufio.NewScanner(strings.NewReader(s))
}

// Path resolves through the adrg/xdg package's own process-wide state,
// which XDG_CONFIG_HOME only refreshes into on an explicit Reload() -
// t.Setenv itself already forbids parallel use, so these two stay
// sequential rather than fighting over that shared state too.
func TestPathIsUnderXDGConfigHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := cliconfig.Path("hush-hush-cli")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "hush-hush-cli", "config.yaml"), path)
}

func TestPathCreatesTheParentDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	_, err := cliconfig.Path("hush-hush-cli")
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(dir, "hush-hush-cli"))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestExistsIsFalseForAMissingFile(t *testing.T) {
	t.Parallel()

	require.False(t, cliconfig.Exists(filepath.Join(t.TempDir(), "config.yaml")))
}

func TestExistsIsTrueForAnExistingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("server: x\n"), 0o600))

	require.True(t, cliconfig.Exists(path))
}

func TestConfirmDefaultsToYesOnBareEnter(t *testing.T) {
	t.Parallel()

	require.True(t, cliconfig.Confirm(scan("\n"), new(strings.Builder), "?"))
}

func TestConfirmAcceptsYVariants(t *testing.T) {
	t.Parallel()

	for _, answer := range []string{"y\n", "Y\n", "yes\n", "YES\n"} {
		require.True(t, cliconfig.Confirm(scan(answer), new(strings.Builder), "?"), "answer %q", answer)
	}
}

func TestConfirmRejectsNVariants(t *testing.T) {
	t.Parallel()

	for _, answer := range []string{"n\n", "N\n", "no\n", "anything-else\n"} {
		require.False(t, cliconfig.Confirm(scan(answer), new(strings.Builder), "?"), "answer %q", answer)
	}
}

func TestConfirmWritesTheQuestion(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	cliconfig.Confirm(scan("y\n"), &out, "write a config now?")

	require.Contains(t, out.String(), "write a config now?")
}

func TestConfirmSharesAScannerWithLaterPrompts(t *testing.T) {
	t.Parallel()

	// The regression this guards: a fresh bufio.Scanner wrapping the same
	// reader can swallow lines a previous scanner already buffered past
	// its own first Scan(). Confirm and PromptValue here share one
	// scanner over input containing both answers up front, the way a
	// real terminal can deliver several already-typed lines in one read.
	sc := scan("y\nhttps://example.com\n")

	require.True(t, cliconfig.Confirm(sc, new(strings.Builder), "?"))

	out, err := cliconfig.PromptValue(sc, new(strings.Builder), "Server", "http://localhost:8080")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", out)
}

func TestShouldWriteStarterSkipsWhenAConfigFileAlreadyExists(t *testing.T) {
	t.Parallel()

	require.False(t, cliconfig.ShouldWriteStarter(true, false, false, true, true))
}

func TestShouldWriteStarterSkipsWhenARelevantEnvVarIsSet(t *testing.T) {
	t.Parallel()

	require.False(t, cliconfig.ShouldWriteStarter(false, true, false, true, true))
}

func TestShouldWriteStarterProceedsOnExplicitYesEvenWithoutATTY(t *testing.T) {
	t.Parallel()

	require.True(t, cliconfig.ShouldWriteStarter(false, false, true, false, false))
}

func TestShouldWriteStarterSkipsNonInteractivelyWithNoExplicitYes(t *testing.T) {
	t.Parallel()

	require.False(t, cliconfig.ShouldWriteStarter(false, false, false, false, true))
}

func TestShouldWriteStarterFollowsTheConfirmationInteractively(t *testing.T) {
	t.Parallel()

	require.True(t, cliconfig.ShouldWriteStarter(false, false, false, true, true))
	require.False(t, cliconfig.ShouldWriteStarter(false, false, false, true, false))
}

func TestResolveSecretReturnsTheLiteralWhenNoCommandIsSet(t *testing.T) {
	t.Parallel()

	out, err := cliconfig.ResolveSecret("literal-value", "")
	require.NoError(t, err)
	assert.Equal(t, "literal-value", out)
}

func TestResolveSecretRunsTheCommandThroughTheShellWhenSet(t *testing.T) {
	t.Parallel()

	out, err := cliconfig.ResolveSecret("ignored-literal", "echo hi | tr a-z A-Z")
	require.NoError(t, err)
	assert.Equal(t, "HI", out)
}

func TestResolveSecretTrimsExactlyOneTrailingNewlineFromTheCommand(t *testing.T) {
	t.Parallel()

	out, err := cliconfig.ResolveSecret("", `printf 'secret\n\n'`)
	require.NoError(t, err)
	assert.Equal(t, "secret\n", out)
}

func TestResolveSecretFailsLoudlyOnANonZeroCommandExit(t *testing.T) {
	t.Parallel()

	_, err := cliconfig.ResolveSecret("", "exit 1")
	require.Error(t, err)
}

func TestPromptValueReturnsCurrentOnBareEnter(t *testing.T) {
	t.Parallel()

	out, err := cliconfig.PromptValue(scan("\n"), new(strings.Builder), "Server", "http://localhost:8080")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", out)
}

func TestPromptValueReturnsTheTypedValue(t *testing.T) {
	t.Parallel()

	out, err := cliconfig.PromptValue(scan("https://example.com\n"), new(strings.Builder), "Server", "http://localhost:8080")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", out)
}

func TestPromptValueShowsTheLabelAndCurrentValue(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	_, err := cliconfig.PromptValue(scan("\n"), &out, "Server URL", "http://localhost:8080")
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Server URL")
	assert.Contains(t, out.String(), "http://localhost:8080")
}

func TestPromptSecretReadsAMaskedLineAndEchoesANewline(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	read := func(int) ([]byte, error) { return []byte("s3cret"), nil }

	value, err := cliconfig.PromptSecret(&out, "Token", 0, read)
	require.NoError(t, err)
	assert.Equal(t, "s3cret", value)
	assert.Contains(t, out.String(), "Token")
}

func TestPromptSecretTrimsSurroundingWhitespace(t *testing.T) {
	t.Parallel()

	read := func(int) ([]byte, error) { return []byte("  s3cret  "), nil }

	value, err := cliconfig.PromptSecret(new(strings.Builder), "Token", 0, read)
	require.NoError(t, err)
	assert.Equal(t, "s3cret", value)
}

func TestPromptSecretPropagatesAReadFailure(t *testing.T) {
	t.Parallel()

	read := func(int) ([]byte, error) { return nil, errNoTTY }

	_, err := cliconfig.PromptSecret(new(strings.Builder), "Token", 0, read)
	require.Error(t, err)
}

func TestPromptPersistenceChoices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		answer     string
		wantChoice cliconfig.PersistChoice
		wantExtra  string
	}{
		{"keyring", "1\n", cliconfig.PersistKeyring, ""},
		{"command", "2\npass show hush-hush/token\n", cliconfig.PersistCommand, "pass show hush-hush/token"},
		{"literal", "3\n", cliconfig.PersistLiteral, ""},
		{"skip on explicit choice", "4\n", cliconfig.PersistSkip, ""},
		{"skip on bare enter", "\n", cliconfig.PersistSkip, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			choice, extra, err := cliconfig.PromptPersistence(scan(tt.answer), new(strings.Builder), "token")
			require.NoError(t, err)
			assert.Equal(t, tt.wantChoice, choice)
			assert.Equal(t, tt.wantExtra, extra)
		})
	}
}

func TestPromptConfigWalksEveryFieldInOrder(t *testing.T) {
	t.Parallel()

	sc := scan(strings.Join([]string{
		"https://example.com", // server
		"caller-id",           // caller
		"age1recipient",       // recipients
	}, "\n") + "\n")

	var out strings.Builder
	readPassword := func(int) ([]byte, error) { return []byte(""), nil } // both credential prompts blank

	result, err := cliconfig.PromptConfig(sc, &out, 0, readPassword, cliconfig.Values{Server: "http://localhost:8080"})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", result.Server)
	assert.Equal(t, "caller-id", result.Caller)
	assert.Equal(t, "age1recipient", result.Recipients)
	assert.Equal(t, cliconfig.PersistSkip, result.Token.Choice)
	assert.Equal(t, cliconfig.PersistSkip, result.Identity.Choice)
}

func TestPromptConfigOffersPersistenceOnlyWhenACredentialValueWasEntered(t *testing.T) {
	t.Parallel()

	// Field order is server, token, caller, recipients, identity (design.md)
	// - the persistence-choice line for each credential interleaves right
	// after that credential's value, not after every plain field.
	sc := scan(strings.Join([]string{
		"",  // server (accept default)
		"3", // token persistence: literal
		"",  // caller
		"",  // recipients
		"3", // identity persistence: literal
	}, "\n") + "\n")

	tokens := []string{"a-token", "an-identity"}
	call := 0
	readPassword := func(int) ([]byte, error) {
		v := tokens[call]
		call++

		return []byte(v), nil
	}

	result, err := cliconfig.PromptConfig(sc, new(strings.Builder), 0, readPassword, cliconfig.Values{Server: "http://localhost:8080"})
	require.NoError(t, err)
	assert.Equal(t, cliconfig.PersistLiteral, result.Token.Choice)
	assert.Equal(t, "a-token", result.Token.Value)
	assert.Equal(t, cliconfig.PersistLiteral, result.Identity.Choice)
	assert.Equal(t, "an-identity", result.Identity.Value)
}
