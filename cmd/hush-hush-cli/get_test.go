package main

import (
	"bytes"
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/seal"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetRunsFromEnvironmentAloneNoFlags mirrors
// TestInjectRunsFromEnvironmentAloneNoFlags - the same "runs unmodified
// inside CI" requirement, for the read path. get needs no bearer token,
// so only HUSH_HUSH_SERVER and HUSH_HUSH_IDENTITY are set.
func TestGetRunsFromEnvironmentAloneNoFlags(t *testing.T) {
	srv, s, _ := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	sealed, err := seal.Seal([]byte("plaintext-value"), []string{identity.Recipient().String()})
	require.NoError(t, err)
	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", sealed, nil, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_IDENTITY", identity.String())

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	var out bytes.Buffer

	root := newRootCmd()
	root.SetArgs([]string{"get", "mattermost_deploy_webhook"})
	root.SetOut(&out)

	require.NoError(t, root.Execute())
	require.Equal(t, "plaintext-value", out.String())
}

// TestGetIdentityCommandWinsOverALiteralIdentity mirrors
// TestConfigTokenCommandWinsOverALiteralToken for the identity/
// identity_command pair (rules/cli.md's "secrets get a command option,
// not just a value" - identity is a credential the same as token).
func TestGetIdentityCommandWinsOverALiteralIdentity(t *testing.T) {
	srv, s, _ := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	sealed, err := seal.Seal([]byte("plaintext-value"), []string{identity.Recipient().String()})
	require.NoError(t, err)
	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", sealed, nil, ""))

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	var out bytes.Buffer

	root := newRootCmd()
	root.SetArgs([]string{
		"get", "mattermost_deploy_webhook",
		"--server", srv.URL,
		"--identity", "not-a-real-identity",
		"--identity-command", "echo " + identity.String(),
	})
	root.SetOut(&out)

	require.NoError(t, root.Execute())
	require.Equal(t, "plaintext-value", out.String())
}

func TestGetFailsFastWithNoIdentityConfigured(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"get", "mattermost_deploy_webhook"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--identity")
	assert.Contains(t, err.Error(), "HUSH_HUSH_IDENTITY")
	assert.Contains(t, err.Error(), "init")
}

func TestGetReportsAnIdentityCommandFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"get", "anything", "--identity-command", "exit 1"})

	err := root.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "identity_command")
}
