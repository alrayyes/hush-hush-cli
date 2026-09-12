package main

import (
	"bytes"
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInjectRunsFromEnvironmentAloneNoFlags is the CLI spec's "runs
// unmodified inside CI" requirement: a CI job supplies configuration
// through its own secret storage as environment variables, never flags,
// and never a bespoke wrapper or Action.
func TestInjectRunsFromEnvironmentAloneNoFlags(t *testing.T) {
	srv, s, token := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("HUSH_HUSH_RECIPIENTS", identity.Recipient().String())

	// A fresh viper instance per test: the package-level default one
	// otherwise carries flag bindings and values across tests in this
	// package, which is exactly what a "no CI-specific code path" test
	// must not rely on to pass.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"inject", "mattermost_deploy_webhook"})
	root.SetIn(bytes.NewReader([]byte("plaintext-value")))

	require.NoError(t, root.Execute())

	obj, err := s.GetObject(t.Context(), "mattermost_deploy_webhook")
	require.NoError(t, err)
	require.NotEqual(t, []byte("plaintext-value"), obj.Value)
}

// TestInjectDescriptionFlagSetsIt drives the real cobra command end to end,
// through the hush-hush-go SDK's regenerated CreateObjectRequest, rather
// than internal/cli.Inject directly.
func TestInjectDescriptionFlagSetsIt(t *testing.T) {
	srv, s, token := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{
		"inject", "mattermost_deploy_webhook",
		"--recipients", identity.Recipient().String(),
		"--description", "prod deploy webhook",
	})
	root.SetIn(bytes.NewReader([]byte("plaintext-value")))

	require.NoError(t, root.Execute())

	obj, err := s.GetObject(t.Context(), "mattermost_deploy_webhook")
	require.NoError(t, err)
	require.Equal(t, "prod deploy webhook", obj.Description)
}

// TestInjectFailsFastWithNoTokenConfigured is rules/cli.md's "commands
// never prompt for a missing value" companion: a missing token fails
// clearly, naming how to fix it, rather than reaching the server with an
// empty bearer token.
func TestInjectFailsFastWithNoTokenConfigured(t *testing.T) {
	srv, s, _ := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_RECIPIENTS", identity.Recipient().String())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"inject", "mattermost_deploy_webhook"})
	root.SetIn(bytes.NewReader([]byte("plaintext-value")))

	err = root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--token")
	assert.Contains(t, err.Error(), "HUSH_HUSH_TOKEN")
	assert.Contains(t, err.Error(), "init")

	_, getErr := s.GetObject(t.Context(), "mattermost_deploy_webhook")
	require.Error(t, getErr) // never created - the server was never called
}

func TestInjectFailsFastWithNoRecipientsConfigured(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"inject", "mattermost_deploy_webhook"})
	root.SetIn(bytes.NewReader([]byte("plaintext-value")))

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--recipients")
	assert.Contains(t, err.Error(), "HUSH_HUSH_RECIPIENTS")
	assert.Contains(t, err.Error(), "init")
}
