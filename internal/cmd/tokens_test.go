package cmd_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenCreateJSONFlagPrintsTheMintedValue(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	root := newRoot(t, srv.URL, token)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"token", "create", "homelab/vps-docker", "--ttl", "1h", "--description", "ci reader", "--json"})

	require.NoError(t, root.Execute())

	var got client.ConsumerTokenWithValue
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, "homelab/vps-docker", got.Consumer)
	assert.Equal(t, "ci reader", got.Description)
	assert.NotEmpty(t, got.Value)
}

func TestTokenCreateRequiresTTL(t *testing.T) {
	t.Parallel()

	srv, _, token := testserver.New(t)

	root := newRoot(t, srv.URL, token)
	root.SetArgs([]string{"token", "create", "homelab/vps-docker"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--ttl")
}

func TestTokenCreateFailsFastWithNoTokenConfigured(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	root := newRoot(t, srv.URL, "")
	root.SetArgs([]string{"token", "create", "homelab/vps-docker", "--ttl", "1h"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--token")
}

func TestTokenListTableIncludesEveryMintedToken(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	s.IssueConsumerToken("homelab/vps-docker", "ci reader", 0)

	root := newRoot(t, srv.URL, token)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"token", "list"})

	require.NoError(t, root.Execute())
	assert.Contains(t, out.String(), "homelab/vps-docker")
	assert.Contains(t, out.String(), "ci reader")
}

func TestTokenRevokeThenPurgeRemovesIt(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	minted := s.IssueConsumerToken("homelab/vps-docker", "", 0)

	revokeRoot := newRoot(t, srv.URL, token)
	revokeRoot.SetArgs([]string{"token", "revoke", minted.ID})
	require.NoError(t, revokeRoot.Execute())

	purgeRoot := newRoot(t, srv.URL, token)
	purgeRoot.SetArgs([]string{"token", "purge", minted.ID})
	require.NoError(t, purgeRoot.Execute())

	require.Empty(t, s.ListConsumerTokens())
}

func TestTokenListShowsStatusNotARevokedBoolean(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	s.IssueConsumerToken("expired-consumer", "", 0)
	s.IssueConsumerToken("active-consumer", "", time.Hour)
	revoked := s.IssueConsumerToken("revoked-consumer", "", time.Hour)
	s.RevokeConsumerToken(revoked.ID)

	root := newRoot(t, srv.URL, token)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"token", "list"})

	require.NoError(t, root.Execute())

	rows := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n")[1:] {
		fields := strings.Fields(line)
		rows[fields[1]] = fields[len(fields)-1]
	}

	assert.Equal(t, "expired", rows["expired-consumer"])
	assert.Equal(t, "active", rows["active-consumer"])
	assert.Equal(t, "revoked", rows["revoked-consumer"])
	assert.NotContains(t, out.String(), "REVOKED")
}

func TestTokenListJSONIncludesStatusAndAllowedActions(t *testing.T) {
	t.Parallel()

	srv, s, token := testserver.New(t)
	s.IssueConsumerToken("active-consumer", "", time.Hour)

	root := newRoot(t, srv.URL, token)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"token", "list", "--json"})

	require.NoError(t, root.Execute())

	var got []map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "active", got[0]["status"])
	assert.ElementsMatch(t, []any{"rotate", "revoke"}, got[0]["allowed_actions"])
}
