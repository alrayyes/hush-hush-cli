package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListRunsFromEnvironmentAloneNoFlags mirrors
// TestDeleteRunsFromEnvironmentAloneNoFlags - the same "runs unmodified
// inside CI" requirement, for the list command.
func TestListRunsFromEnvironmentAloneNoFlags(t *testing.T) {
	srv, s, token := testserver.New(t)

	require.NoError(t, s.CreateObject(t.Context(), "zebra", []byte("sealed"), []string{"homelab/vps-docker"}, "z desc"))
	require.NoError(t, s.CreateObject(t.Context(), "apple", []byte("sealed"), nil, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"list"})

	require.NoError(t, root.Execute())

	stdout := out.String()
	assert.Contains(t, stdout, "apple")
	assert.Contains(t, stdout, "zebra")
	assert.Contains(t, stdout, "homelab/vps-docker")
	assert.Contains(t, stdout, "z desc")
	assert.Less(t, strings.Index(stdout, "apple"), strings.Index(stdout, "zebra"), "want apple sorted before zebra")
}

// TestListFailsFastWithNoTokenConfigured mirrors
// TestDeleteFailsFastWithNoTokenConfigured for the list command: listing
// is gated by the write token, same as delete/update, unlike get.
func TestListFailsFastWithNoTokenConfigured(t *testing.T) {
	srv, _, _ := testserver.New(t)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"list"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--token")
	assert.Contains(t, err.Error(), "HUSH_HUSH_TOKEN")
	assert.Contains(t, err.Error(), "init")
}

func TestListJSONFlagPrintsRawArray(t *testing.T) {
	srv, s, token := testserver.New(t)

	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", []byte("sealed"), []string{"homelab/vps-docker"}, "deploy hook"))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"list", "--json"})

	require.NoError(t, root.Execute())

	var got []struct {
		ID          string   `json:"id"`
		UsedBy      []string `json:"used_by"`
		Description string   `json:"description"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "mattermost_deploy_webhook", got[0].ID)
	assert.Equal(t, []string{"homelab/vps-docker"}, got[0].UsedBy)
	assert.Equal(t, "deploy hook", got[0].Description)
}

func TestListEmptyStorePrintsJustTheHeader(t *testing.T) {
	srv, _, token := testserver.New(t)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"list"})

	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "ID")
	assert.Contains(t, out.String(), "USED BY")
	assert.Contains(t, out.String(), "DESCRIPTION")
}
