package main

import (
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestDeleteRunsFromEnvironmentAloneNoFlags mirrors
// TestInjectRunsFromEnvironmentAloneNoFlags - the same "runs unmodified
// inside CI" requirement, for the delete command.
func TestDeleteRunsFromEnvironmentAloneNoFlags(t *testing.T) {
	srv, s, token := testserver.New(t)

	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", []byte("sealed"), nil, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"delete", "mattermost_deploy_webhook"})

	require.NoError(t, root.Execute())

	_, err := s.GetObject(t.Context(), "mattermost_deploy_webhook")
	require.ErrorIs(t, err, testserver.ErrNotFound)
}
