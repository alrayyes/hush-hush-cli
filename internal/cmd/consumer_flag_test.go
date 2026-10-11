package cmd_test

import (
	"bytes"
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/seal"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// twoVariants (variant_id_test.go) stores release_token for homelab/a and
// homelab/b, "first" and "second".

func TestGetConsumerReadsTheVariantThatNamesIt(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"get", "release_token", "--consumer", "homelab/b", "--identity", identity.String()})

	require.NoError(t, root.Execute())
	assert.Equal(t, "second", out.String())
}

func TestUpdateConsumerChangesOnlyThatVariant(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	first, second := twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	root.SetIn(bytes.NewReader([]byte("rotated")))
	root.SetArgs([]string{"update", "release_token", "--consumer", "homelab/b", "--recipients", identity.Recipient().String()})

	require.NoError(t, root.Execute())

	plain := func(variantID string) string {
		obj, getErr := s.GetVariant(t.Context(), "release_token", variantID)
		require.NoError(t, getErr)

		value, openErr := seal.Unseal(obj.Value, []string{identity.String()})
		require.NoError(t, openErr)

		return string(value)
	}

	assert.Equal(t, "first", plain(first))
	assert.Equal(t, "rotated", plain(second))
}

func TestDeleteConsumerRemovesOnlyThatVariant(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	first, second := twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"delete", "release_token", "--consumer", "homelab/b"})

	require.NoError(t, root.Execute())

	_, err = s.GetVariant(t.Context(), "release_token", first)
	require.NoError(t, err)

	_, err = s.GetVariant(t.Context(), "release_token", second)
	require.ErrorIs(t, err, testserver.ErrNotFound)
}

func TestUsedByConsumerListsThatVariantsConsumers(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"used-by", "release_token", "--consumer", "homelab/b"})

	require.NoError(t, root.Execute())
	assert.Equal(t, "homelab/b\n", out.String())
}

func TestConsumerThatNoVariantNamesIsNotFound(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"get", "release_token", "--consumer", "homelab/nobody", "--identity", identity.String()})

	require.ErrorContains(t, root.Execute(), "not found")
}

func TestConsumerAndIDTogetherFailBeforeAnyRequest(t *testing.T) {
	t.Parallel()

	// An address nothing listens on: the command must fail on the flags
	// alone, never reaching the network.
	root := newRoot(t, "http://127.0.0.1:1", "token")
	root.SetArgs([]string{"delete", "release_token", "--id", "00000000-0000-4000-8000-000000000000", "--consumer", "homelab/b"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--id")
	assert.Contains(t, err.Error(), "--consumer")
	assert.NotContains(t, err.Error(), "connect")
}
