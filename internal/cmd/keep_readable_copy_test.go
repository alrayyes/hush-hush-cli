package cmd_test

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/cmd"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runWrite(t *testing.T, srv *httptest.Server, token string, args ...string) error {
	t.Helper()

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("HUSH_HUSH_TOKEN", token)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := cmd.NewRootCmd("dev")
	root.SetArgs(args)
	root.SetIn(bytes.NewReader([]byte("plaintext-value")))

	if err := root.Execute(); err != nil {
		return err //nolint:wrapcheck // the test asserts on the command's own error
	}

	return nil
}

func decryptsWith(t *testing.T, sealed []byte, identity age.Identity) bool {
	t.Helper()

	r, err := age.Decrypt(bytes.NewReader(sealed), identity)
	if err != nil {
		return false
	}

	plaintext, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, []byte("plaintext-value"), plaintext)

	return true
}

func ownerWithKey(t *testing.T, s *testserver.Store) *age.X25519Identity {
	t.Helper()

	owner, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	s.SetOwnerPublicKey(owner.Recipient().String())

	return owner
}

func TestInjectKeepReadableCopySealsToConsumerAndOwner(t *testing.T) {
	srv, s, token := testserver.New(t)
	owner := ownerWithKey(t, s)
	consumer := newConsumerWithKey(t, s, "c")

	require.NoError(t, runWrite(t, srv, token, "inject", "secret", "--used-by", "c", "--keep-readable-copy"))

	obj, err := s.GetObject(t.Context(), "secret")
	require.NoError(t, err)
	assert.True(t, decryptsWith(t, obj.Value, consumer), "consumer can decrypt")
	assert.True(t, decryptsWith(t, obj.Value, owner), "owner can decrypt")
}

func TestInjectWithoutKeepReadableCopyNeverAddsTheOwner(t *testing.T) {
	srv, s, token := testserver.New(t)
	owner := ownerWithKey(t, s)
	newConsumerWithKey(t, s, "c")

	require.NoError(t, runWrite(t, srv, token, "inject", "secret", "--used-by", "c"))

	obj, err := s.GetObject(t.Context(), "secret")
	require.NoError(t, err)
	assert.False(t, decryptsWith(t, obj.Value, owner))
}

func TestUpdateKeepReadableCopySealsToOwnerToo(t *testing.T) {
	srv, s, token := testserver.New(t)
	owner := ownerWithKey(t, s)
	recipient, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	require.NoError(t, s.CreateObject(t.Context(), "secret", []byte("old"), nil, ""))

	require.NoError(t, runWrite(t, srv, token, "update", "secret", "--recipients", recipient.Recipient().String(), "--keep-readable-copy"))

	obj, err := s.GetObject(t.Context(), "secret")
	require.NoError(t, err)
	assert.True(t, decryptsWith(t, obj.Value, recipient))
	assert.True(t, decryptsWith(t, obj.Value, owner))
}

func TestKeepReadableCopyWithNoOwnerKeyFailsBeforeWriting(t *testing.T) {
	srv, s, token := testserver.New(t)
	newConsumerWithKey(t, s, "c")

	err := runWrite(t, srv, token, "inject", "secret", "--used-by", "c", "--keep-readable-copy")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escrowed")

	_, getErr := s.GetObject(t.Context(), "secret")
	require.Error(t, getErr, "never created")
}
