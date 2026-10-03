package seal_test

import (
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/seal"
	"github.com/stretchr/testify/require"
)

// FuzzUnseal feeds Unseal arbitrary bytes as the "sealed" value and an
// arbitrary string as the identity: what the server stores and what a
// config file holds are both input this package doesn't control. Either
// may fail; neither may panic. The seed corpus (these f.Add calls) runs in
// every plain `go test`; `go test -fuzz=FuzzUnseal` explores beyond it.
func FuzzUnseal(f *testing.F) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(f, err)

	sealed, err := seal.Seal([]byte("a secret"), []string{identity.Recipient().String()})
	require.NoError(f, err)

	f.Add(sealed, identity.String())
	f.Add(sealed[:len(sealed)/2], identity.String()) // truncated mid-stream
	f.Add(sealed[:10], identity.String())            // truncated inside the header
	f.Add([]byte{}, identity.String())
	f.Add([]byte("age-encryption.org/v1\n"), identity.String()) // header, no recipient
	f.Add([]byte("plaintext, not an age file"), identity.String())
	f.Add(sealed, "")
	f.Add(sealed, "AGE-SECRET-KEY-1")
	f.Add(sealed, "not a key")

	f.Fuzz(func(_ *testing.T, data []byte, key string) {
		_, _ = seal.Unseal(data, []string{key})
	})
}

// FuzzSeal feeds Seal arbitrary recipient strings, which come straight from
// --recipients, HUSH_HUSH_RECIPIENTS or a consumer's registered key. A
// recipient that parses must produce a value its identity can open again.
func FuzzSeal(f *testing.F) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(f, err)

	f.Add([]byte("a secret"), identity.Recipient().String())
	f.Add([]byte{}, identity.Recipient().String())
	f.Add([]byte("a secret"), "")
	f.Add([]byte("a secret"), "age1")
	f.Add([]byte("a secret"), "not a recipient")
	f.Add([]byte("a secret"), identity.String()) // a private key where a public one belongs

	f.Fuzz(func(t *testing.T, value []byte, recipient string) {
		sealed, err := seal.Seal(value, []string{recipient})
		if err != nil {
			return
		}

		// Only the generated recipient has an identity to open it with.
		if recipient != identity.Recipient().String() {
			return
		}

		got, err := seal.Unseal(sealed, []string{identity.String()})
		require.NoError(t, err)
		require.Equal(t, value, got)
	})
}
