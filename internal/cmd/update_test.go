package cmd_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/cmd"
	"github.com/alrayyes/hush-hush-cli/internal/seal"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateRunsFromEnvironmentAloneNoFlags mirrors
// TestInjectRunsFromEnvironmentAloneNoFlags - the same "runs unmodified
// inside CI" requirement, for the write-path update command.
func TestUpdateRunsFromEnvironmentAloneNoFlags(t *testing.T) {
	srv, s, token := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	sealed, err := seal.Seal([]byte("old-value"), []string{identity.Recipient().String()})
	require.NoError(t, err)
	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", sealed, nil, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("HUSH_HUSH_RECIPIENTS", identity.Recipient().String())

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cmd.NewRootCmd("dev")
	root.SetArgs([]string{"update", "mattermost_deploy_webhook"})
	root.SetIn(bytes.NewReader([]byte("new-value")))

	require.NoError(t, root.Execute())

	obj, err := s.GetObject(t.Context(), "mattermost_deploy_webhook")
	require.NoError(t, err)
	require.NotEqual(t, []byte("new-value"), obj.Value)
}

// TestUpdateFailsFastWithNoTokenConfigured mirrors
// TestInjectFailsFastWithNoTokenConfigured for the update command.
func TestUpdateFailsFastWithNoTokenConfigured(t *testing.T) {
	srv, s, _ := testserver.New(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	sealed, err := seal.Seal([]byte("old-value"), []string{identity.Recipient().String()})
	require.NoError(t, err)
	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", sealed, nil, ""))

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_RECIPIENTS", identity.Recipient().String())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cmd.NewRootCmd("dev")
	root.SetArgs([]string{"update", "mattermost_deploy_webhook"})
	root.SetIn(bytes.NewReader([]byte("new-value")))

	err = root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--token")
	assert.Contains(t, err.Error(), "HUSH_HUSH_TOKEN")
	assert.Contains(t, err.Error(), "init")

	// Unchanged - the request never reached the server.
	obj, err := s.GetObject(t.Context(), "mattermost_deploy_webhook")
	require.NoError(t, err)
	require.NotEqual(t, []byte("new-value"), obj.Value)
}

func newConsumerWithKey(t *testing.T, s *testserver.Store, name string) *age.X25519Identity {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	require.NoError(t, s.AddConsumer(t.Context(), name))

	recipient := identity.Recipient().String()
	_, err = s.UpdateConsumer(t.Context(), name, nil, &recipient)
	require.NoError(t, err)

	return identity
}

func runUpdate(t *testing.T, srv *httptest.Server, token string, args ...string) error {
	t.Helper()

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := cmd.NewRootCmd("dev")
	root.SetArgs(append([]string{"update", "secret"}, args...))
	root.SetIn(bytes.NewReader([]byte("new-value")))

	if err := root.Execute(); err != nil {
		return fmt.Errorf("run update: %w", err)
	}

	return nil
}

func TestUpdateUsedByReplacesConsumersAndSealsToTheirKey(t *testing.T) {
	srv, s, token := testserver.New(t)
	b := newConsumerWithKey(t, s, "b")

	require.NoError(t, s.CreateObject(t.Context(), "secret", []byte("old"), []string{"a"}, ""))

	require.NoError(t, runUpdate(t, srv, token, "--used-by", "b"))

	obj, err := s.GetObject(t.Context(), "secret")
	require.NoError(t, err)
	assert.Equal(t, []string{"b"}, obj.UsedBy)

	r, err := age.Decrypt(bytes.NewReader(obj.Value), b)
	require.NoError(t, err)

	plaintext, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, []byte("new-value"), plaintext)
}

func TestUpdateWithoutUsedByLeavesConsumersAlone(t *testing.T) {
	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	require.NoError(t, s.CreateObject(t.Context(), "secret", []byte("old"), []string{"a"}, ""))

	require.NoError(t, runUpdate(t, srv, token, "--recipients", identity.Recipient().String()))

	obj, err := s.GetObject(t.Context(), "secret")
	require.NoError(t, err)
	assert.Equal(t, []string{"a"}, obj.UsedBy)
}

func TestUpdateClearUsedByRemovesEveryConsumer(t *testing.T) {
	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	require.NoError(t, s.CreateObject(t.Context(), "secret", []byte("old"), []string{"a"}, ""))

	require.NoError(t, runUpdate(t, srv, token, "--clear-used-by", "--recipients", identity.Recipient().String()))

	obj, err := s.GetObject(t.Context(), "secret")
	require.NoError(t, err)
	assert.Empty(t, obj.UsedBy)
}

func TestUpdateUsedByAndClearUsedByCantBeCombined(t *testing.T) {
	srv, s, token := testserver.New(t)
	require.NoError(t, s.CreateObject(t.Context(), "secret", []byte("old"), nil, ""))

	err := runUpdate(t, srv, token, "--used-by", "b", "--clear-used-by")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--clear-used-by")
}

func TestUpdateUsedByWithNoRegisteredKeyFailsAndLeavesTheObjectAlone(t *testing.T) {
	srv, s, token := testserver.New(t)
	require.NoError(t, s.AddConsumer(t.Context(), "b"))
	require.NoError(t, s.CreateObject(t.Context(), "secret", []byte("old"), []string{"a"}, ""))

	err := runUpdate(t, srv, token, "--used-by", "b")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "b")

	obj, getErr := s.GetObject(t.Context(), "secret")
	require.NoError(t, getErr)
	assert.Equal(t, []string{"a"}, obj.UsedBy)
	assert.Equal(t, []byte("old"), obj.Value)
}
