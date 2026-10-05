package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The real server caps a list request that carries no paging parameter at
// one page of 50 (alrayyes/Hush-Hush#662). testserver does the same, so
// these tests only pass when the SDK underneath pages through limit/offset
// and X-Total-Count, which hush-hush-go does from v4.4.0.
const beyondOnePage = 120

func listJSON(t *testing.T, server, token string, args ...string) []map[string]any {
	t.Helper()

	root := newRoot(t, server, token)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs(append(args, "--json"))

	require.NoError(t, root.Execute())

	var rows []map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &rows))

	return rows
}

func TestListShowsEveryObjectPastTheServersDefaultPage(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)

	for i := range beyondOnePage {
		require.NoError(t, s.CreateObject(t.Context(), fmt.Sprintf("secret-%03d", i), []byte("sealed"), nil, ""))
	}

	assert.Equal(t, beyondOnePage, len(listJSON(t, srv.URL, token, "list"))) //nolint:testifylint // assert.Len would print all 120 rows on failure
}

func TestTokenListShowsEveryTokenPastTheServersDefaultPage(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)

	for i := range beyondOnePage {
		s.CreateConsumerToken(fmt.Sprintf("consumer-%03d", i))
	}

	assert.Equal(t, beyondOnePage, len(listJSON(t, srv.URL, token, "token", "list"))) //nolint:testifylint // assert.Len would print all 120 rows on failure
}
