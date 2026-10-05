package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/cmd"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runConsumerCmd(t *testing.T, srv *httptest.Server, token string, args ...string) (string, error) {
	t.Helper()

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := cmd.NewRootCmd("dev")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs(append([]string{"consumer"}, args...))

	err := root.Execute()

	return out.String(), err
}

func TestConsumerListTableShowsNameCountAndKey(t *testing.T) {
	srv, s, token := testserver.New(t)
	require.NoError(t, s.AddConsumer(t.Context(), "homelab/vps"))
	_, err := s.UpdateConsumer(t.Context(), "homelab/vps", nil, new("age1abc"))
	require.NoError(t, err)

	out, err := runConsumerCmd(t, srv, token, "list")

	require.NoError(t, err)
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "homelab/vps")
	assert.Contains(t, out, "age1abc")
}

func TestConsumerListJSONAndQuery(t *testing.T) {
	srv, s, token := testserver.New(t)
	require.NoError(t, s.AddConsumer(t.Context(), "homelab/vps"))
	require.NoError(t, s.AddConsumer(t.Context(), "ci/runner"))

	out, err := runConsumerCmd(t, srv, token, "list", "--query", "ci", "--json")

	require.NoError(t, err)

	var got []client.Consumer
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "ci/runner", got[0].Name)
}

func TestConsumerAddThenAddAgainFails(t *testing.T) {
	srv, _, token := testserver.New(t)

	_, err := runConsumerCmd(t, srv, token, "add", "homelab/new")
	require.NoError(t, err)

	_, err = runConsumerCmd(t, srv, token, "add", "homelab/new")
	require.ErrorIs(t, err, client.ErrConsumerAlreadyExists)
}

func TestConsumerUpdateRenamesAndSetsTheKey(t *testing.T) {
	srv, s, token := testserver.New(t)
	require.NoError(t, s.AddConsumer(t.Context(), "old"))

	out, err := runConsumerCmd(t, srv, token, "update", "old", "--name", "fresh", "--public-key", "age1xyz", "--json")

	require.NoError(t, err)

	var got client.Consumer
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, client.Consumer{Name: "fresh", PublicKey: "age1xyz"}, got)
}

func TestConsumerUpdateNeedsAtLeastOneFlag(t *testing.T) {
	srv, _, token := testserver.New(t)

	_, err := runConsumerCmd(t, srv, token, "update", "old")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--name")
}

func TestConsumerUpdateUnknownRenameFails(t *testing.T) {
	srv, _, token := testserver.New(t)

	_, err := runConsumerCmd(t, srv, token, "update", "ghost", "--name", "other")

	require.ErrorIs(t, err, client.ErrConsumerNotFound)
}

func TestConsumerDeleteRemovesIt(t *testing.T) {
	srv, s, token := testserver.New(t)
	require.NoError(t, s.AddConsumer(t.Context(), "gone"))

	_, err := runConsumerCmd(t, srv, token, "delete", "gone")
	require.NoError(t, err)

	out, err := runConsumerCmd(t, srv, token, "list", "--json")
	require.NoError(t, err)
	assert.JSONEq(t, "[]", out)
}

func TestConsumerAddFailsFastWithNoTokenConfigured(t *testing.T) {
	srv, _, _ := testserver.New(t)

	_, err := runConsumerCmd(t, srv, "", "add", "x")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--token")
}
