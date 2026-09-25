package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUsedByRunsFromEnvironmentAloneNoFlags mirrors
// TestGetRunsFromEnvironmentAloneNoFlags: used-by needs no bearer token,
// so only HUSH_HUSH_SERVER is set.
func TestUsedByRunsFromEnvironmentAloneNoFlags(t *testing.T) {
	srv, s, _ := testserver.New(t)

	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", []byte("sealed"),
		[]string{"homelab/vps-docker"}, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"used-by", "mattermost_deploy_webhook"})

	require.NoError(t, root.Execute())
	assert.Equal(t, "homelab/vps-docker\n", out.String())
}

func TestUsedByJSONFlagPrintsRawArray(t *testing.T) {
	srv, s, _ := testserver.New(t)

	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", []byte("sealed"),
		[]string{"homelab/vps-docker"}, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"used-by", "mattermost_deploy_webhook", "--json"})

	require.NoError(t, root.Execute())

	var got []string
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, []string{"homelab/vps-docker"}, got)
}

func TestUsedByNoConsumersPrintsNoLines(t *testing.T) {
	srv, s, _ := testserver.New(t)

	require.NoError(t, s.CreateObject(t.Context(), "x", []byte("sealed"), nil, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"used-by", "x"})

	require.NoError(t, root.Execute())
	assert.Empty(t, out.String())
}

func TestUsedByUnknownIDFails(t *testing.T) {
	srv, _, _ := testserver.New(t)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"used-by", "nope"})

	err := root.Execute()
	require.Error(t, err)
}
