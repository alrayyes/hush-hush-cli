package cmd_test

import (
	"bytes"
	"io"
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/cmd"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
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

	// Every NewRootCmd owns a fresh viper, so no flag binding or value
	// carries over from another test: exactly what a "no CI-specific code
	// path" test must not rely on to pass.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cmd.NewRootCmd("dev")
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
	t.Parallel()

	srv, s, token := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	root := newRoot(t, srv.URL, token)
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

	root := cmd.NewRootCmd("dev")
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

	root := cmd.NewRootCmd("dev")
	root.SetArgs([]string{"inject", "mattermost_deploy_webhook"})
	root.SetIn(bytes.NewReader([]byte("plaintext-value")))

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--recipients")
	assert.Contains(t, err.Error(), "HUSH_HUSH_RECIPIENTS")
	assert.Contains(t, err.Error(), "--used-by")
	assert.Contains(t, err.Error(), "init")
}

// TestInjectWithUsedByAndNoRecipientsResolvesTheConsumersRegisteredKey
// drives the real cobra command end to end: no --recipients flag at all,
// just --used-by naming a consumer already registered with a public key
// (issue #125).
func TestInjectWithUsedByAndNoRecipientsResolvesTheConsumersRegisteredKey(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	require.NoError(t, s.AddConsumer(t.Context(), "homelab/vps-docker"))
	recipient := identity.Recipient().String()
	_, err = s.UpdateConsumer(t.Context(), "homelab/vps-docker", nil, &recipient)
	require.NoError(t, err)

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"inject", "mattermost_deploy_webhook", "--used-by", "homelab/vps-docker"})
	root.SetIn(bytes.NewReader([]byte("plaintext-value")))

	require.NoError(t, root.Execute())

	obj, err := s.GetObject(t.Context(), "mattermost_deploy_webhook")
	require.NoError(t, err)

	r, err := age.Decrypt(bytes.NewReader(obj.Value), identity)
	require.NoError(t, err)

	plaintext, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, []byte("plaintext-value"), plaintext)
}

// TestInjectWithUsedByAndNoRegisteredKeyFailsNamingTheConsumer is issue
// #125's own acceptance criteria: a --used-by consumer with no registered
// key must fail clearly, not seal to fewer recipients than requested.
func TestInjectWithUsedByAndNoRegisteredKeyFailsNamingTheConsumer(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)

	require.NoError(t, s.AddConsumer(t.Context(), "homelab/vps-docker"))

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"inject", "mattermost_deploy_webhook", "--used-by", "homelab/vps-docker"})
	root.SetIn(bytes.NewReader([]byte("plaintext-value")))

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "homelab/vps-docker")

	_, getErr := s.GetObject(t.Context(), "mattermost_deploy_webhook")
	require.Error(t, getErr, "never created - resolution must fail before the server is called")
}
