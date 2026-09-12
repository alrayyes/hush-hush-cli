package main

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigTokenCommandWinsOverALiteralToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	root := newRootCmd()
	require.NoError(t, root.PersistentFlags().Set("server", "http://localhost:8080"))
	require.NoError(t, root.PersistentFlags().Set("token", "literal-token"))
	require.NoError(t, root.PersistentFlags().Set("token-command", "echo command-token"))

	cfg, err := config()
	require.NoError(t, err)
	assert.Equal(t, "command-token", cfg.Token)
}

func TestConfigReportsATokenCommandFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	root := newRootCmd()
	require.NoError(t, root.PersistentFlags().Set("server", "http://localhost:8080"))
	require.NoError(t, root.PersistentFlags().Set("token-command", "exit 1"))

	_, err := config()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token_command")
}
