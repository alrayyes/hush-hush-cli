package client_test

import (
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/require"
)

func TestListConsumersReturnsEveryEntryWithItsCountAndKey(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)
	require.NoError(t, store.AddConsumer(t.Context(), "alpha"))
	_, err := store.UpdateConsumer(t.Context(), "alpha", nil, new("age1abc"))
	require.NoError(t, err)
	require.NoError(t, store.AddConsumer(t.Context(), "beta"))

	cl, err := client.New(srv.URL, token)
	require.NoError(t, err)

	got, err := cl.ListConsumers(t.Context(), "")
	require.NoError(t, err)
	require.Equal(t, []client.Consumer{
		{Name: "alpha", PublicKey: "age1abc"},
		{Name: "beta"},
	}, got)
}

func TestListConsumersFollowsEveryPage(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)
	for i := range 120 {
		require.NoError(t, store.AddConsumer(t.Context(), "c"+string(rune('a'+i/26))+string(rune('a'+i%26))))
	}

	cl, err := client.New(srv.URL, token)
	require.NoError(t, err)

	got, err := cl.ListConsumers(t.Context(), "")
	require.NoError(t, err)
	require.Len(t, got, 120)
}

func TestListConsumersFiltersByQuery(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)
	require.NoError(t, store.AddConsumer(t.Context(), "homelab/vps"))
	require.NoError(t, store.AddConsumer(t.Context(), "ci/runner"))

	cl, err := client.New(srv.URL, token)
	require.NoError(t, err)

	got, err := cl.ListConsumers(t.Context(), "HOME")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "homelab/vps", got[0].Name)
}

func TestAddConsumerCreatesAnEntryWithNoSecrets(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	cl, err := client.New(srv.URL, token)
	require.NoError(t, err)

	got, err := cl.AddConsumer(t.Context(), "homelab/new")
	require.NoError(t, err)
	require.Equal(t, client.Consumer{Name: "homelab/new"}, got)
}

func TestAddConsumerTwiceFailsWithConsumerAlreadyExists(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)
	require.NoError(t, store.AddConsumer(t.Context(), "dup"))

	cl, err := client.New(srv.URL, token)
	require.NoError(t, err)

	_, err = cl.AddConsumer(t.Context(), "dup")
	require.ErrorIs(t, err, client.ErrConsumerAlreadyExists)
}

func TestUpdateConsumerRenamesAndRegistersAKey(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)
	require.NoError(t, store.AddConsumer(t.Context(), "old"))

	cl, err := client.New(srv.URL, token)
	require.NoError(t, err)

	got, err := cl.UpdateConsumer(t.Context(), "old", new("fresh"), new("age1xyz"))
	require.NoError(t, err)
	require.Equal(t, client.Consumer{Name: "fresh", PublicKey: "age1xyz"}, got)
}

func TestUpdateConsumerRenamingAnUnknownNameFails(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	cl, err := client.New(srv.URL, token)
	require.NoError(t, err)

	_, err = cl.UpdateConsumer(t.Context(), "ghost", new("other"), nil)
	require.ErrorIs(t, err, client.ErrConsumerNotFound)
}

func TestDeleteConsumerRemovesItFromTheDirectory(t *testing.T) {
	t.Parallel()

	srv, store, token := testserver.New(t)
	require.NoError(t, store.AddConsumer(t.Context(), "gone"))

	cl, err := client.New(srv.URL, token)
	require.NoError(t, err)

	require.NoError(t, cl.DeleteConsumer(t.Context(), "gone"))

	got, err := cl.ListConsumers(t.Context(), "")
	require.NoError(t, err)
	require.Empty(t, got)
}
