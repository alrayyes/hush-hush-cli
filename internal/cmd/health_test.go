package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// health is unauthenticated like status: a probe with no token still gets
// an answer.
func TestHealthRunsWithNoTokenConfigured(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	root := newRoot(t, srv.URL, "")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"health"})

	require.NoError(t, root.Execute())
	assert.Equal(t, "status: ok\n", out.String())
}

func TestHealthPrintsTheEnvironmentLabelWhenTheServerSetsOne(t *testing.T) {
	t.Parallel()

	srv, store, _ := testserver.New(t)
	store.SetEnvironment("prod / homelab")

	root := newRoot(t, srv.URL, "")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"health"})

	require.NoError(t, root.Execute())
	assert.Equal(t, "status: ok\nenvironment: prod / homelab\n", out.String())
}

func TestHealthJSONFlagPrintsRawObject(t *testing.T) {
	t.Parallel()

	srv, store, _ := testserver.New(t)
	store.SetEnvironment("staging")

	root := newRoot(t, srv.URL, "")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"health", "--json"})

	require.NoError(t, root.Execute())

	var got struct {
		Status      string `json:"status"`
		Environment string `json:"environment"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, "ok", got.Status)
	assert.Equal(t, "staging", got.Environment)
}

func TestHealthFailsWhenTheServerIsDown(t *testing.T) {
	t.Parallel()

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(down.Close)

	root := newRoot(t, down.URL, "")
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetOut(&bytes.Buffer{})
	root.SetArgs([]string{"health"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "health")
}
