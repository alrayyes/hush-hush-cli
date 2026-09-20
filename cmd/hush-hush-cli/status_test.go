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

// TestStatusRunsWithNoTokenConfigured is the same "unauthenticated by
// design" check the SDK's own AuthStatus test makes, at the command level:
// a CI job with no HUSH_HUSH_TOKEN set at all still gets a real answer
// rather than the --token nudge list/inject/update/delete give.
func TestStatusRunsWithNoTokenConfigured(t *testing.T) {
	srv, _, _ := testserver.New(t)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"status"})

	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "bootstrapped: true")
}

// TestStatusReportsAnUnbootstrappedServerAsSuccess is #107's own
// acceptance criterion: an unbootstrapped server is a normal, exit-0
// result to report, not an error to surface as one.
func TestStatusReportsAnUnbootstrappedServerAsSuccess(t *testing.T) {
	srv, store, _ := testserver.New(t)
	store.SetBootstrapped(false)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"status"})

	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "bootstrapped: false")
}

func TestStatusJSONFlagPrintsRawObject(t *testing.T) {
	srv, store, _ := testserver.New(t)
	store.SetBootstrapped(false)

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"status", "--json"})

	require.NoError(t, root.Execute())

	var got struct {
		Bootstrapped bool `json:"bootstrapped"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.False(t, got.Bootstrapped)
}
