package cli_test

import (
	"context"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/stretchr/testify/require"
)

func TestListReturnsEveryObjectSortedByID(t *testing.T) {
	t.Parallel()

	srv, s, token := newTestServer(t)

	require.NoError(t, s.CreateObject(t.Context(), "zebra", []byte("v"), []string{"homelab/vps-docker"}, "z desc"))
	require.NoError(t, s.CreateObject(t.Context(), "apple", []byte("v"), nil, ""))

	cfg := cli.Config{Server: srv.URL, Token: token}
	objects, err := cli.List(context.Background(), cfg)
	require.NoError(t, err)
	require.Len(t, objects, 2)

	require.Equal(t, "apple", objects[0].ID)
	require.Equal(t, "zebra", objects[1].ID)
	require.Equal(t, []string{"homelab/vps-docker"}, objects[1].UsedBy)
	require.Equal(t, "z desc", objects[1].Description)
}

func TestListWithoutAValidTokenIsRejected(t *testing.T) {
	t.Parallel()

	srv, _, _ := newTestServer(t)

	cfg := cli.Config{Server: srv.URL, Token: "wrong-token"}
	_, err := cli.List(context.Background(), cfg)
	require.ErrorIs(t, err, client.ErrUnauthorized)
}

func TestListEmptyStoreReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)

	cfg := cli.Config{Server: srv.URL, Token: token}
	objects, err := cli.List(context.Background(), cfg)
	require.NoError(t, err)
	require.Empty(t, objects)
}
