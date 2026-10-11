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

// twoVariants stores RELEASE_TOKEN twice, "first" for homelab/a and "second"
// for homelab/b, both sealed to identity, and returns their UUIDs in that
// order.
func twoVariants(t *testing.T, s *testserver.Store, identity *age.X25519Identity) (first, second string) {
	t.Helper()

	recipients := []string{identity.Recipient().String()}

	sealedFirst, err := seal.Seal([]byte("first"), recipients)
	require.NoError(t, err)
	require.NoError(t, s.CreateObject(t.Context(), "release_token", sealedFirst, []string{"homelab/a"}, ""))

	sealedSecond, err := seal.Seal([]byte("second"), recipients)
	require.NoError(t, err)

	second, err = s.AddVariant(t.Context(), "release_token", sealedSecond, []string{"homelab/b"})
	require.NoError(t, err)

	listed, err := s.ListObjects(t.Context(), "homelab/a")
	require.NoError(t, err)
	require.Len(t, listed, 1)

	return listed[0].ID, second
}

func TestGetIDReadsThatVariant(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	_, second := twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"get", "release_token", "--id", second, "--identity", identity.String()})

	require.NoError(t, root.Execute())
	assert.Equal(t, "second", out.String())
}

func TestGetWithoutIDOnSeveralVariantsSaysToPassOne(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"get", "release_token", "--identity", identity.String()})

	err = root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "several variants")
	assert.Contains(t, err.Error(), "--id")
}

func TestGetIDOfAnotherNameFails(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"get", "release_token", "--id", "00000000-0000-4000-8000-000000000000", "--identity", identity.String()})

	require.ErrorContains(t, root.Execute(), "not found")
}

func TestUpdateIDChangesOnlyThatVariant(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	first, second := twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	root.SetIn(bytes.NewReader([]byte("rotated")))
	root.SetArgs([]string{"update", "release_token", "--id", second, "--recipients", identity.Recipient().String()})

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

func TestDeleteIDRemovesOnlyThatVariant(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	first, second := twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"delete", "release_token", "--id", second})

	require.NoError(t, root.Execute())

	_, err = s.GetVariant(t.Context(), "release_token", first)
	require.NoError(t, err)

	_, err = s.GetVariant(t.Context(), "release_token", second)
	require.ErrorIs(t, err, testserver.ErrNotFound)
}

func TestUsedByIDListsThatVariantsConsumers(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	_, second := twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"used-by", "release_token", "--id", second})

	require.NoError(t, root.Execute())
	assert.Equal(t, "homelab/b\n", out.String())
}
