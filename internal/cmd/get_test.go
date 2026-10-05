package cmd_test

import (
	"bytes"
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/cmd"
	"github.com/alrayyes/hush-hush-cli/internal/seal"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetRunsFromEnvironmentAloneNoFlags mirrors
// TestInjectRunsFromEnvironmentAloneNoFlags - the same "runs unmodified
// inside CI" requirement, for the read path. A write token still
// authorizes get (alrayyes/hush-hush#446), so HUSH_HUSH_TOKEN is set
// alongside HUSH_HUSH_SERVER and HUSH_HUSH_IDENTITY.
func TestGetRunsFromEnvironmentAloneNoFlags(t *testing.T) {
	srv, s, token := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	sealed, err := seal.Seal([]byte("plaintext-value"), []string{identity.Recipient().String()})
	require.NoError(t, err)
	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", sealed, nil, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("HUSH_HUSH_IDENTITY", identity.String())

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	var out bytes.Buffer

	root := cmd.NewRootCmd("dev")
	root.SetArgs([]string{"get", "mattermost_deploy_webhook"})
	root.SetOut(&out)

	require.NoError(t, root.Execute())
	require.Equal(t, "plaintext-value", out.String())
}

// TestGetUsesConsumerTokenFromEnvironmentWhenNoWriteToken exercises this
// change's feature through the full CLI surface (cobra flags, viper env
// binding, config()) rather than internal/cli.Get directly - the other
// tests for the fallback behavior itself live in internal/cli/get_test.go.
func TestGetUsesConsumerTokenFromEnvironmentWhenNoWriteToken(t *testing.T) {
	srv, s, _ := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	sealed, err := seal.Seal([]byte("plaintext-value"), []string{identity.Recipient().String()})
	require.NoError(t, err)
	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", sealed, []string{"homelab/vps-docker"}, ""))

	consumerToken := s.CreateConsumerToken("homelab/vps-docker")

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_CONSUMER_TOKEN", consumerToken)
	t.Setenv("HUSH_HUSH_IDENTITY", identity.String())

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	var out bytes.Buffer

	root := cmd.NewRootCmd("dev")
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
	srv, s, token := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	sealed, err := seal.Seal([]byte("plaintext-value"), []string{identity.Recipient().String()})
	require.NoError(t, err)
	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", sealed, nil, ""))

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	var out bytes.Buffer

	root := cmd.NewRootCmd("dev")
	root.SetArgs([]string{
		"get", "mattermost_deploy_webhook",
		"--server", srv.URL,
		"--token", token,
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

	root := cmd.NewRootCmd("dev")
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

	root := cmd.NewRootCmd("dev")
	root.SetArgs([]string{"get", "anything", "--identity-command", "exit 1"})

	err := root.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "identity_command")
}
