package cli_test

import (
	"context"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/stretchr/testify/require"
)

func TestUsedByReturnsAnObjectsRecordedConsumers(t *testing.T) {
	t.Parallel()

	srv, s, _ := newTestServer(t)

	require.NoError(t, s.CreateObject(t.Context(), "mattermost_deploy_webhook", []byte("v"),
		[]string{"homelab/vps-docker"}, ""))

	cfg := cli.Config{Server: srv.URL}
	usedBy, err := cli.UsedBy(context.Background(), cfg, "mattermost_deploy_webhook")
	require.NoError(t, err)
	require.Equal(t, []string{"homelab/vps-docker"}, usedBy)
}

func TestUsedByNeedsNoToken(t *testing.T) {
	t.Parallel()

	srv, s, _ := newTestServer(t)

	require.NoError(t, s.CreateObject(t.Context(), "x", []byte("v"), nil, ""))

	cfg := cli.Config{Server: srv.URL}
	_, err := cli.UsedBy(context.Background(), cfg, "x")
	require.NoError(t, err)
}

func TestUsedByUnknownIDReturnsNotFound(t *testing.T) {
	t.Parallel()

	srv, _, _ := newTestServer(t)

	cfg := cli.Config{Server: srv.URL}
	_, err := cli.UsedBy(context.Background(), cfg, "nope")
	require.ErrorIs(t, err, client.ErrNotFound)
}
