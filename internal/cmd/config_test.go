package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigTokenCommandWinsOverALiteralToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	root, v := newRootCmd("dev")
	require.NoError(t, root.PersistentFlags().Set("server", "http://localhost:8080"))
	require.NoError(t, root.PersistentFlags().Set("token", "literal-token"))
	require.NoError(t, root.PersistentFlags().Set("token-command", "echo command-token"))

	cfg, err := config(v, false)
	require.NoError(t, err)
	assert.Equal(t, "command-token", cfg.Token)
}

func TestConfigReportsATokenCommandFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	root, v := newRootCmd("dev")
	require.NoError(t, root.PersistentFlags().Set("server", "http://localhost:8080"))
	require.NoError(t, root.PersistentFlags().Set("token-command", "exit 1"))

	_, err := config(v, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token_command")
}

// TestConfigConsumerTokenCommandWinsOverALiteralConsumerToken mirrors
// TestConfigTokenCommandWinsOverALiteralToken for consumer_token/
// consumer_token_command - the same precedent, per this change's design.md.
func TestConfigConsumerTokenCommandWinsOverALiteralConsumerToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	root, v := newRootCmd("dev")
	require.NoError(t, root.PersistentFlags().Set("server", "http://localhost:8080"))
	require.NoError(t, root.PersistentFlags().Set("consumer-token", "literal-consumer-token"))
	require.NoError(t, root.PersistentFlags().Set("consumer-token-command", "echo command-consumer-token"))

	cfg, err := config(v, false)
	require.NoError(t, err)
	assert.Equal(t, "command-consumer-token", cfg.ConsumerToken)
}

// TestConfigConsumerTokenResolvesFromEveryLayer covers task 2.1's
// verification: flag > env > config file precedence, for consumer_token
// specifically - mirroring how token already resolves from each layer,
// per this change's cli-config delta ("Configuration precedence").
func TestConfigConsumerTokenResolvesFromEveryLayer(t *testing.T) {
	t.Run("config file", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)

		path := filepath.Join(dir, "hush-hush-cli", "config.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte("server: http://localhost:8080\nconsumer_token: file-consumer-token\n"), 0o600))

		_, v := newRootCmd("dev") // registers flags and reads the config file into its viper

		cfg, err := config(v, false)
		require.NoError(t, err)
		assert.Equal(t, "file-consumer-token", cfg.ConsumerToken)
	})

	t.Run("env overrides config file", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)

		path := filepath.Join(dir, "hush-hush-cli", "config.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte("server: http://localhost:8080\nconsumer_token: file-consumer-token\n"), 0o600))
		t.Setenv("HUSH_HUSH_CONSUMER_TOKEN", "env-consumer-token")

		_, v := newRootCmd("dev")

		cfg, err := config(v, false)
		require.NoError(t, err)
		assert.Equal(t, "env-consumer-token", cfg.ConsumerToken)
	})

	t.Run("flag overrides env and config file", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)

		path := filepath.Join(dir, "hush-hush-cli", "config.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte("server: http://localhost:8080\nconsumer_token: file-consumer-token\n"), 0o600))
		t.Setenv("HUSH_HUSH_CONSUMER_TOKEN", "env-consumer-token")

		root, v := newRootCmd("dev")
		require.NoError(t, root.PersistentFlags().Set("consumer-token", "flag-consumer-token"))

		cfg, err := config(v, false)
		require.NoError(t, err)
		assert.Equal(t, "flag-consumer-token", cfg.ConsumerToken)
	})
}

func TestConfigReportsAConsumerTokenCommandFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	root, v := newRootCmd("dev")
	require.NoError(t, root.PersistentFlags().Set("server", "http://localhost:8080"))
	require.NoError(t, root.PersistentFlags().Set("consumer-token-command", "exit 1"))

	_, err := config(v, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "consumer_token_command")
}
