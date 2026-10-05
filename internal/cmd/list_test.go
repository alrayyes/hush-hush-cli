package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/cmd"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
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

	root := cmd.NewRootCmd("dev")
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

	root := cmd.NewRootCmd("dev")
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

	root := cmd.NewRootCmd("dev")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"list", "--json"})

	require.NoError(t, root.Execute())

	var got []struct {
		Slug        string   `json:"slug"`
		UsedBy      []string `json:"used_by"`
		Description string   `json:"description"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "mattermost_deploy_webhook", got[0].Slug)
	assert.Equal(t, []string{"homelab/vps-docker"}, got[0].UsedBy)
	assert.Equal(t, "deploy hook", got[0].Description)
}

func TestListEmptyStorePrintsJustTheHeader(t *testing.T) {
	srv, _, token := testserver.New(t)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cmd.NewRootCmd("dev")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"list"})

	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "ID")
	assert.Contains(t, out.String(), "USED BY")
	assert.Contains(t, out.String(), "DESCRIPTION")
}

func TestListUsedByFlagFiltersToThatConsumer(t *testing.T) {
	srv, s, token := testserver.New(t)

	require.NoError(t, s.CreateObject(t.Context(), "for-a", []byte("sealed"), []string{"a"}, ""))
	require.NoError(t, s.CreateObject(t.Context(), "for-b", []byte("sealed"), []string{"b"}, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cmd.NewRootCmd("dev")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"list", "--used-by", "a", "--json"})

	require.NoError(t, root.Execute())

	assert.Contains(t, out.String(), "for-a")
	assert.NotContains(t, out.String(), "for-b")
}

func TestListTableShowsCreatedAndUpdatedTimes(t *testing.T) {
	srv, s, token := testserver.New(t)

	require.NoError(t, s.CreateObject(t.Context(), "apple", []byte("sealed"), nil, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cmd.NewRootCmd("dev")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"list"})

	require.NoError(t, root.Execute())

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "CREATED")
	assert.Contains(t, lines[0], "UPDATED")
	assert.Contains(t, lines[1], time.Now().Format(time.DateOnly))
}

func TestListJSONIncludesCreatedUpdatedAndActors(t *testing.T) {
	srv, s, token := testserver.New(t)

	require.NoError(t, s.CreateObject(t.Context(), "apple", []byte("sealed"), nil, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cmd.NewRootCmd("dev")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"list", "--json"})

	require.NoError(t, root.Execute())

	var got []map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got, 1)

	for _, key := range []string{"created_at", "updated_at", "created_by", "updated_by"} {
		assert.Contains(t, got[0], key)
	}
}

func listSlugs(t *testing.T, srv *httptest.Server, token string, args ...string) []string {
	t.Helper()

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cmd.NewRootCmd("dev")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs(append([]string{"list", "--json"}, args...))

	require.NoError(t, root.Execute())

	var got []struct {
		Slug string `json:"slug"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))

	slugs := make([]string, len(got))
	for i, o := range got {
		slugs[i] = o.Slug
	}

	return slugs
}

func seedTaggedObjects(t *testing.T, s *testserver.Store) {
	t.Helper()

	require.NoError(t, s.CreateObject(t.Context(), "a", []byte("v"), []string{"c"}, ""))
	require.NoError(t, s.CreateObject(t.Context(), "b", []byte("v"), []string{"d"}, ""))
	require.NoError(t, s.CreateObject(t.Context(), "c", []byte("v"), []string{"c"}, ""))
	require.NoError(t, s.SetObjectTags(t.Context(), "a", []string{"prod", "ci"}))
	require.NoError(t, s.SetObjectTags(t.Context(), "b", []string{"staging"}))
	require.NoError(t, s.SetObjectTags(t.Context(), "c", []string{"prod"}))
}

func TestListTagFiltersToObjectsCarryingIt(t *testing.T) {
	srv, s, token := testserver.New(t)
	seedTaggedObjects(t, s)

	assert.Equal(t, []string{"a", "c"}, listSlugs(t, srv, token, "--tag", "prod"))
}

func TestListRepeatedAndCommaSeparatedTagsMustAllMatch(t *testing.T) {
	srv, s, token := testserver.New(t)
	seedTaggedObjects(t, s)

	assert.Equal(t, []string{"a"}, listSlugs(t, srv, token, "--tag", "prod", "--tag", "ci"))
	assert.Equal(t, []string{"a"}, listSlugs(t, srv, token, "--tag", "prod,ci"))
}

func TestListTagCombinesWithUsedByAndIgnoresCase(t *testing.T) {
	srv, s, token := testserver.New(t)
	seedTaggedObjects(t, s)

	assert.Equal(t, []string{"b"}, listSlugs(t, srv, token, "--tag", "Staging", "--used-by", "d"))
	assert.Empty(t, listSlugs(t, srv, token, "--tag", "prod", "--used-by", "d"))
	assert.Equal(t, []string{"a", "c"}, listSlugs(t, srv, token, "--tag", "PROD"))
}

func TestListWithoutTagStillListsEverything(t *testing.T) {
	srv, s, token := testserver.New(t)
	seedTaggedObjects(t, s)

	assert.Equal(t, []string{"a", "b", "c"}, listSlugs(t, srv, token))
}
