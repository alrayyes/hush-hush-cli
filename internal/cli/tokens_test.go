package cli_test

import (
	"context"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/stretchr/testify/require"
)

// injectForToken stores a secret sealed to a fresh identity, used_by
// consumer - the fixture every consumer-token lifecycle test starts from
// to prove a minted token actually authorizes (or stops authorizing) a
// real Get.
func injectForToken(t *testing.T, srv, token, consumer string) (id string, identity *age.X25519Identity, value []byte) {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	value = []byte("plaintext-value")

	cfg := cli.Config{Server: srv, Token: token}
	require.NoError(t, cli.Inject(context.Background(), cfg, "x", value,
		[]string{identity.Recipient().String()}, []string{consumer}, ""))

	return "x", identity, value
}

func TestCreateConsumerTokenMintsAUsableReadToken(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)
	id, identity, value := injectForToken(t, srv.URL, token, "homelab/vps-docker")

	cfg := cli.Config{Server: srv.URL, Token: token, RequireToken: true}
	minted, err := cli.CreateConsumerToken(context.Background(), cfg, "homelab/vps-docker", "ci reader", time.Hour)
	require.NoError(t, err)
	require.Equal(t, "homelab/vps-docker", minted.Consumer)
	require.NotEmpty(t, minted.Value)

	getCfg := cli.Config{Server: srv.URL, ConsumerToken: minted.Value}
	plaintext, err := cli.Get(context.Background(), getCfg, id, []string{identity.String()})
	require.NoError(t, err)
	require.Equal(t, value, plaintext)
}

func TestCreateConsumerTokenWithoutAValidTokenIsRejected(t *testing.T) {
	t.Parallel()

	srv, _, _ := newTestServer(t)

	cfg := cli.Config{Server: srv.URL, Token: "wrong-token", RequireToken: true}
	_, err := cli.CreateConsumerToken(context.Background(), cfg, "homelab/vps-docker", "", time.Hour)
	require.ErrorIs(t, err, client.ErrUnauthorized)
}

func TestListConsumerTokensReturnsEveryMintedToken(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)

	cfg := cli.Config{Server: srv.URL, Token: token, RequireToken: true}
	_, err := cli.CreateConsumerToken(context.Background(), cfg, "homelab/vps-docker", "ci reader", time.Hour)
	require.NoError(t, err)

	tokens, err := cli.ListConsumerTokens(context.Background(), cfg)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.Equal(t, "homelab/vps-docker", tokens[0].Consumer)
	require.Equal(t, "ci reader", tokens[0].Description)
}

func TestRotateConsumerTokenReplacesTheValue(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)
	id, identity, value := injectForToken(t, srv.URL, token, "homelab/vps-docker")

	cfg := cli.Config{Server: srv.URL, Token: token, RequireToken: true}
	minted, err := cli.CreateConsumerToken(context.Background(), cfg, "homelab/vps-docker", "", time.Hour)
	require.NoError(t, err)

	rotated, err := cli.RotateConsumerToken(context.Background(), cfg, minted.ID, time.Hour)
	require.NoError(t, err)
	require.Equal(t, minted.ID, rotated.ID)
	require.NotEqual(t, minted.Value, rotated.Value)

	oldTokenCfg := cli.Config{Server: srv.URL, ConsumerToken: minted.Value}
	_, err = cli.Get(context.Background(), oldTokenCfg, id, []string{identity.String()})
	require.ErrorIs(t, err, client.ErrUnauthorized, "the old value must stop authenticating once rotated")

	newTokenCfg := cli.Config{Server: srv.URL, ConsumerToken: rotated.Value}
	plaintext, err := cli.Get(context.Background(), newTokenCfg, id, []string{identity.String()})
	require.NoError(t, err)
	require.Equal(t, value, plaintext)
}

func TestRotateConsumerTokenUnknownIDReturnsNotFound(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)

	cfg := cli.Config{Server: srv.URL, Token: token, RequireToken: true}
	_, err := cli.RotateConsumerToken(context.Background(), cfg, "nope", time.Hour)
	require.ErrorIs(t, err, client.ErrConsumerTokenNotFound)
}

func TestRevokeConsumerTokenInvalidatesIt(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)
	id, identity, _ := injectForToken(t, srv.URL, token, "homelab/vps-docker")

	cfg := cli.Config{Server: srv.URL, Token: token, RequireToken: true}
	minted, err := cli.CreateConsumerToken(context.Background(), cfg, "homelab/vps-docker", "", time.Hour)
	require.NoError(t, err)

	require.NoError(t, cli.RevokeConsumerToken(context.Background(), cfg, minted.ID))

	getCfg := cli.Config{Server: srv.URL, ConsumerToken: minted.Value}
	_, err = cli.Get(context.Background(), getCfg, id, []string{identity.String()})
	require.ErrorIs(t, err, client.ErrUnauthorized)
}

func TestRevokeConsumerTokenUnknownIDIsNotAnError(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)

	cfg := cli.Config{Server: srv.URL, Token: token, RequireToken: true}
	require.NoError(t, cli.RevokeConsumerToken(context.Background(), cfg, "nope"))
}

func TestPurgeConsumerTokenRemovesARevokedToken(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)

	cfg := cli.Config{Server: srv.URL, Token: token, RequireToken: true}
	minted, err := cli.CreateConsumerToken(context.Background(), cfg, "homelab/vps-docker", "", time.Hour)
	require.NoError(t, err)
	require.NoError(t, cli.RevokeConsumerToken(context.Background(), cfg, minted.ID))

	require.NoError(t, cli.PurgeConsumerToken(context.Background(), cfg, minted.ID))

	tokens, err := cli.ListConsumerTokens(context.Background(), cfg)
	require.NoError(t, err)
	require.Empty(t, tokens)
}

func TestPurgeConsumerTokenStillActiveIsRejected(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)

	cfg := cli.Config{Server: srv.URL, Token: token, RequireToken: true}
	minted, err := cli.CreateConsumerToken(context.Background(), cfg, "homelab/vps-docker", "", time.Hour)
	require.NoError(t, err)

	err = cli.PurgeConsumerToken(context.Background(), cfg, minted.ID)
	require.ErrorIs(t, err, client.ErrConsumerTokenActive)
}
