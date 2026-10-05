package cliconfig_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/cliconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errUnwritable = errors.New("unwritable")

// unsetEnv clears every HUSH_HUSH_* variable for the test, so the machine
// running it doesn't count as already configured.
func unsetEnv(t *testing.T) {
	t.Helper()

	for _, name := range cliconfig.EnvVars {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
}

func offer(t *testing.T, in string, mutate func(*cliconfig.OfferOptions)) (wrote bool, path string, out, errOut string, err error) {
	t.Helper()

	path = filepath.Join(t.TempDir(), "config.yaml")

	var outBuf, errBuf bytes.Buffer

	opts := cliconfig.OfferOptions{
		ResolvePath:  func() (string, error) { return path, nil },
		In:           strings.NewReader(in),
		Out:          &outBuf,
		Err:          &errBuf,
		ReadPassword: func(int) ([]byte, error) { return []byte(""), nil },
	}
	if mutate != nil {
		mutate(&opts)
	}

	wrote, err = cliconfig.OfferInit(opts)

	return wrote, path, outBuf.String(), errBuf.String(), err
}

// Not parallel in any of these: unsetEnv uses t.Setenv.

func TestOfferInitSkipsAndNeverResolvesAPathWhenTheEnvironmentIsConfigured(t *testing.T) {
	t.Setenv("HUSH_HUSH_SERVER", "http://x")

	resolved := false
	wrote, _, _, errOut, err := offer(t, "", func(o *cliconfig.OfferOptions) {
		o.ResolvePath = func() (string, error) {
			resolved = true

			return "", errUnwritable
		}
	})

	require.NoError(t, err)
	assert.False(t, wrote)
	assert.False(t, resolved, "resolving a path creates a directory, which can fail on a CI runner")
	assert.Empty(t, errOut)
}

func TestOfferInitIsAdvisoryWhenThePathCannotBeResolved(t *testing.T) {
	unsetEnv(t)

	wrote, _, _, _, err := offer(t, "", func(o *cliconfig.OfferOptions) {
		o.ResolvePath = func() (string, error) { return "", errUnwritable }
	})

	require.NoError(t, err)
	assert.False(t, wrote)
}

func TestOfferInitLeavesAnExistingConfigAlone(t *testing.T) {
	unsetEnv(t)

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("server: kept\n"), 0o600))

	wrote, _, _, errOut, err := offer(t, "", func(o *cliconfig.OfferOptions) {
		o.Yes = true
		o.ResolvePath = func() (string, error) { return path, nil }
	})

	require.NoError(t, err)
	assert.False(t, wrote)
	assert.Empty(t, errOut)
	assert.Equal(t, "server: kept\n", readFile(t, path))
}

func TestOfferInitPrintsANudgeWhenNonInteractiveAndUnconfigured(t *testing.T) {
	unsetEnv(t)

	wrote, path, _, errOut, err := offer(t, "", nil)

	require.NoError(t, err)
	assert.False(t, wrote)
	assert.Contains(t, errOut, "hush-hush-cli init")
	assert.NoFileExists(t, path)
}

func TestOfferInitYesWritesTheStarterTemplateWithNoPrompt(t *testing.T) {
	unsetEnv(t)

	wrote, path, out, _, err := offer(t, "", func(o *cliconfig.OfferOptions) { o.Yes = true })

	require.NoError(t, err)
	assert.True(t, wrote)
	assert.Empty(t, out, "no prompt under --yes")
	assert.Equal(t, cliconfig.StarterConfig, readFile(t, path))
}

func TestOfferInitInteractiveYesRunsThePromptsOnTheSameScanner(t *testing.T) {
	unsetEnv(t)

	// One scanner must serve the confirm and every prompt after it: the
	// confirm answer, server, caller and recipients, in one reader.
	wrote, path, out, errOut, err := offer(t, "y\nhttps://example.com\nme\nage1abc\n", func(o *cliconfig.OfferOptions) {
		o.Interactive = true
	})

	require.NoError(t, err)
	assert.True(t, wrote)
	assert.Contains(t, out, "No config file found. Set one up now?")
	assert.Empty(t, errOut)

	content := readFile(t, path)
	assert.Contains(t, content, `server: "https://example.com"`)
	assert.Contains(t, content, `caller: "me"`)
	assert.Contains(t, content, `recipients: "age1abc"`)
}

func TestOfferInitInteractiveNoWritesNothingAndPrintsNoNudge(t *testing.T) {
	unsetEnv(t)

	wrote, path, _, errOut, err := offer(t, "n\n", func(o *cliconfig.OfferOptions) { o.Interactive = true })

	require.NoError(t, err)
	assert.False(t, wrote)
	assert.Empty(t, errOut)
	assert.NoFileExists(t, path)
}
