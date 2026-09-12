package main

import (
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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

// TestDeleteFailsFastWithNoTokenConfigured mirrors
// TestInjectFailsFastWithNoTokenConfigured for the delete command.
func TestDeleteFailsFastWithNoTokenConfigured(t *testing.T) {
	srv, s, _ := testserver.New(t)

	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", []byte("sealed"), nil, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"delete", "mattermost_deploy_webhook"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--token")
	assert.Contains(t, err.Error(), "HUSH_HUSH_TOKEN")
	assert.Contains(t, err.Error(), "init")

	// Still there - the request never reached the server.
	_, getErr := s.GetObject(t.Context(), "mattermost_deploy_webhook")
	require.NoError(t, getErr)
}
