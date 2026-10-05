package cmd_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/cmd"
	"github.com/alrayyes/hush-hush-cli/internal/seal"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tagsEnv sets process-wide environment (see helpers_test.go), so the tests
// that call it are not parallel. It points the CLI at srv with one age recipient, and returns that
// recipient so update/inject can seal.
func tagsEnv(t *testing.T, srvURL, token string) string {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	t.Setenv("HUSH_HUSH_SERVER", srvURL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("HUSH_HUSH_RECIPIENTS", identity.Recipient().String())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	return identity.Recipient().String()
}

func runWithStdin(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()

	root := cmd.NewRootCmd("dev")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetIn(bytes.NewReader([]byte(stdin)))
	root.SetArgs(args)

	err := root.Execute()

	return out.String(), err
}

func TestInjectTagFlagSetsTagsThatListShows(t *testing.T) {
	srv, _, token := testserver.New(t)
	tagsEnv(t, srv.URL, token)

	_, err := runWithStdin(t, "v", "inject", "db", "--tag", "prod", "--tag", "CI")
	require.NoError(t, err)

	out, err := runWithStdin(t, "", "list", "--json")
	require.NoError(t, err)

	var got []client.ObjectMetadata
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	require.Len(t, got, 1)
	assert.Equal(t, []string{"prod", "ci"}, got[0].Tags)
}

func TestListTableHasATagsColumn(t *testing.T) {
	srv, _, token := testserver.New(t)
	tagsEnv(t, srv.URL, token)

	_, err := runWithStdin(t, "v", "inject", "db", "--tag", "prod,ci")
	require.NoError(t, err)

	out, err := runWithStdin(t, "", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "TAGS")
	assert.Contains(t, out, "prod,ci")
}

func TestInjectInvalidTagShowsTheServersMessage(t *testing.T) {
	srv, _, token := testserver.New(t)
	tagsEnv(t, srv.URL, token)

	_, err := runWithStdin(t, "v", "inject", "db", "--tag", "has space")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tag")
}

func updateSetup(t *testing.T) (*testserver.Store, string) {
	t.Helper()

	srv, s, token := testserver.New(t)
	recipient := tagsEnv(t, srv.URL, token)

	sealed, err := seal.Seal([]byte("old"), []string{recipient})
	require.NoError(t, err)
	require.NoError(t, s.CreateObject(t.Context(), "db", sealed, nil, ""))
	require.NoError(t, s.SetObjectTags(t.Context(), "db", []string{"prod"}))

	return s, "db"
}

func TestUpdateTagFlagReplacesTags(t *testing.T) {
	s, id := updateSetup(t)

	_, err := runWithStdin(t, "new", "update", id, "--tag", "staging")
	require.NoError(t, err)

	obj, err := s.GetObject(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, []string{"staging"}, obj.Tags)
}

func TestUpdateClearTagsEmptiesThem(t *testing.T) {
	s, id := updateSetup(t)

	_, err := runWithStdin(t, "new", "update", id, "--clear-tags")
	require.NoError(t, err)

	obj, err := s.GetObject(t.Context(), id)
	require.NoError(t, err)
	assert.Empty(t, obj.Tags)
}

func TestUpdateWithNeitherFlagLeavesTagsUntouched(t *testing.T) {
	s, id := updateSetup(t)

	_, err := runWithStdin(t, "new", "update", id)
	require.NoError(t, err)

	obj, err := s.GetObject(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, []string{"prod"}, obj.Tags)
}

func TestUpdateTagAndClearTagsTogetherIsRejected(t *testing.T) {
	_, id := updateSetup(t)

	_, err := runWithStdin(t, "new", "update", id, "--tag", "x", "--clear-tags")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tag")
}
