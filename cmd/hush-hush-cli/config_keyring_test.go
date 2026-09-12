package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// TestConfigKeyringGetPrintsAStoredValue doesn't run t.Parallel(): MockInit
// swaps the package-level keyring backend for an in-memory fake, which
// isn't safe to share with a concurrently-running test that expects the
// real one.
func TestConfigKeyringGetPrintsAStoredValue(t *testing.T) {
	keyring.MockInit()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	viper.Reset()

	require.NoError(t, keyring.Set(keyringService, "token", "s3cret"))

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"config", "keyring-get", "token"})
	require.NoError(t, root.Execute())

	assert.Equal(t, "s3cret\n", out.String())
}

func TestConfigKeyringGetFailsForAMissingValue(t *testing.T) {
	keyring.MockInit()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"config", "keyring-get", "token"})

	require.Error(t, root.Execute())
}
