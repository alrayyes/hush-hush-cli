package cmd_test

import (
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/seal"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedObject(t *testing.T, s *testserver.Store, id, description string) []byte {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	sealed, err := seal.Seal([]byte("the-value"), []string{identity.Recipient().String()})
	require.NoError(t, err)
	require.NoError(t, s.CreateObject(t.Context(), id, sealed, []string{"homelab/a"}, description))

	return sealed
}

func TestEditChangesOnlyTheGivenFieldsAndReadsNoStdin(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	sealed := seedObject(t, s, "db_password", "old")
	require.NoError(t, s.SetObjectTags(t.Context(), "db_password", []string{"prod"}))

	root := newRoot(t, srv.URL, token)
	root.SetIn(errReader{})
	root.SetArgs([]string{"edit", "db_password", "--description", "the primary database"})

	require.NoError(t, root.Execute())

	obj, err := s.GetObject(t.Context(), "db_password")
	require.NoError(t, err)
	assert.Equal(t, "the primary database", obj.Description)
	assert.Equal(t, []string{"prod"}, obj.Tags, "tags were not given, so they stay")
	assert.Equal(t, sealed, obj.Value, "the sealed value is never touched")
}

func TestEditTagReplacesAndClearTagsRemoves(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	seedObject(t, s, "db_password", "")

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"edit", "db_password", "--tag", "Staging,ci"})
	require.NoError(t, root.Execute())

	obj, err := s.GetObject(t.Context(), "db_password")
	require.NoError(t, err)
	assert.Equal(t, []string{"staging", "ci"}, obj.Tags)

	root = newRoot(t, srv.URL, token)
	root.SetArgs([]string{"edit", "db_password", "--clear-tags"})
	require.NoError(t, root.Execute())

	obj, err = s.GetObject(t.Context(), "db_password")
	require.NoError(t, err)
	assert.Empty(t, obj.Tags)
}

func TestEditSlugRenamesTheSecret(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	seedObject(t, s, "db_password", "")

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"edit", "db_password", "--slug", "primary_db_password"})
	require.NoError(t, root.Execute())

	_, err := s.GetObject(t.Context(), "db_password")
	require.ErrorIs(t, err, testserver.ErrNotFound)

	_, err = s.GetObject(t.Context(), "primary_db_password")
	require.NoError(t, err)
}

func TestEditSlugCollisionShowsTheServersRefusalAndChangesNothing(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	seedObject(t, s, "db_password", "")
	seedObject(t, s, "other", "")

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"edit", "db_password", "--slug", "other"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	_, err = s.GetObject(t.Context(), "db_password")
	require.NoError(t, err, "a refused rename leaves the original in place")
}

func TestEditWithoutAFieldFlagIsAUsageError(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	seedObject(t, s, "db_password", "keep")

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"edit", "db_password"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--description")

	obj, err := s.GetObject(t.Context(), "db_password")
	require.NoError(t, err)
	assert.Equal(t, "keep", obj.Description)
}

func TestEditTagWithClearTagsIsRefused(t *testing.T) {
	t.Parallel()

	root := newBareRoot(t)
	root.SetArgs([]string{"edit", "x", "--tag", "a", "--clear-tags"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--tag and --clear-tags")
}

func TestEditPicksTheVariantByConsumerOrID(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	first, second := twoVariants(t, s, identity)

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"edit", "release_token", "--description", "no variant picked"})
	err = root.Execute()
	require.Error(t, err, "a name with several variants needs --id or --consumer")

	root = newRoot(t, srv.URL, token)
	root.SetArgs([]string{"edit", "release_token", "--id", second, "--description", "b side"})
	require.NoError(t, root.Execute())

	root = newRoot(t, srv.URL, token)
	root.SetArgs([]string{"edit", "release_token", "--consumer", "homelab/a", "--description", "a side"})
	require.NoError(t, root.Execute())

	a, err := s.GetVariant(t.Context(), "release_token", first)
	require.NoError(t, err)
	assert.Equal(t, "a side", a.Description)

	b, err := s.GetVariant(t.Context(), "release_token", second)
	require.NoError(t, err)
	assert.Equal(t, "b side", b.Description)
}

func TestEditIDWithConsumerIsRefused(t *testing.T) {
	t.Parallel()

	root := newBareRoot(t)
	root.SetArgs([]string{"edit", "x", "--description", "d", "--id", "i", "--consumer", "c"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--id and --consumer")
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { panic("edit must not read stdin") }
