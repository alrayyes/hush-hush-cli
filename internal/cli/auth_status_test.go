package cli_test

import (
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/stretchr/testify/require"
)

func TestAuthStatusReturnsTheServersBootstrappedValue(t *testing.T) {
	t.Parallel()

	srv, store, _ := newTestServer(t)
	store.SetBootstrapped(false)

	cfg := cli.Config{Server: srv.URL}
	status, err := cli.AuthStatus(t.Context(), cfg)
	require.NoError(t, err)
	require.False(t, status.Bootstrapped)
}
