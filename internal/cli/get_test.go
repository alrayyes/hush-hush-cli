package cli_test

import (
	"context"
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/stretchr/testify/require"
)

func TestGetDecryptsTheStoredValue(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	value := []byte("plaintext-value")

	injectCfg := cli.Config{Server: srv.URL, Token: token}
	require.NoError(t, cli.Inject(context.Background(), injectCfg, "mattermost_deploy_webhook", value,
		[]string{identity.Recipient().String()}, nil, ""))

	getCfg := cli.Config{Server: srv.URL, Token: token}
	plaintext, err := cli.Get(context.Background(), getCfg, "mattermost_deploy_webhook", []string{identity.String()})
	require.NoError(t, err)
	require.Equal(t, value, plaintext)
}

func TestGetWithNoMatchingIdentityFailsClearly(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)

	sealedTo, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	wrongIdentity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	injectCfg := cli.Config{Server: srv.URL, Token: token}
	require.NoError(t, cli.Inject(context.Background(), injectCfg, "x", []byte("v"),
		[]string{sealedTo.Recipient().String()}, nil, ""))

	getCfg := cli.Config{Server: srv.URL, Token: token}
	_, err = cli.Get(context.Background(), getCfg, "x", []string{wrongIdentity.String()})
	require.Error(t, err)
}

func TestGetUnknownIDReturnsNotFound(t *testing.T) {
	t.Parallel()

	srv, _, token := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	getCfg := cli.Config{Server: srv.URL, Token: token}
	_, err = cli.Get(context.Background(), getCfg, "nope", []string{identity.String()})
	require.ErrorIs(t, err, client.ErrNotFound)
}

// TestGetFallsBackToConsumerTokenWhenNoWriteTokenConfigured is this
// change's central behavior (design.md's "Decisions", cli-config delta's
// "Consumer read token used as a fallback on object fetches"): a caller
// with no write token, but with a consumer token scoped to an object's
// used_by, can still read it.
func TestGetFallsBackToConsumerTokenWhenNoWriteTokenConfigured(t *testing.T) {
	t.Parallel()

	srv, s, token := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	value := []byte("plaintext-value")

	injectCfg := cli.Config{Server: srv.URL, Token: token}
	require.NoError(t, cli.Inject(context.Background(), injectCfg, "x", value,
		[]string{identity.Recipient().String()}, []string{"homelab/vps-docker"}, ""))

	consumerToken := s.CreateConsumerToken("homelab/vps-docker")

	getCfg := cli.Config{Server: srv.URL, ConsumerToken: consumerToken}
	plaintext, err := cli.Get(context.Background(), getCfg, "x", []string{identity.String()})
	require.NoError(t, err)
	require.Equal(t, value, plaintext)
}

// TestGetPrefersWriteTokenOverConsumerToken confirms the write token wins
// when both are configured - a consumer token bound to an unrelated
// consumer would otherwise reject a read the write token alone allows.
func TestGetPrefersWriteTokenOverConsumerToken(t *testing.T) {
	t.Parallel()

	srv, s, token := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	value := []byte("plaintext-value")

	injectCfg := cli.Config{Server: srv.URL, Token: token}
	require.NoError(t, cli.Inject(context.Background(), injectCfg, "x", value,
		[]string{identity.Recipient().String()}, nil, ""))

	wrongScopeToken := s.CreateConsumerToken("unrelated-consumer")

	getCfg := cli.Config{Server: srv.URL, Token: token, ConsumerToken: wrongScopeToken}
	plaintext, err := cli.Get(context.Background(), getCfg, "x", []string{identity.String()})
	require.NoError(t, err)
	require.Equal(t, value, plaintext)
}

// TestGetWithConsumerTokenScopedToADifferentConsumerReturnsNotFound
// covers design.md's Risks/Trade-offs: a valid consumer token bound to a
// consumer the object's used_by doesn't include is indistinguishable
// from an unknown object, not a 401/403 - matching api/openapi.yaml's
// anti-enumeration rule on the real endpoint.
func TestGetWithConsumerTokenScopedToADifferentConsumerReturnsNotFound(t *testing.T) {
	t.Parallel()

	srv, s, token := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	injectCfg := cli.Config{Server: srv.URL, Token: token}
	require.NoError(t, cli.Inject(context.Background(), injectCfg, "x", []byte("v"),
		[]string{identity.Recipient().String()}, []string{"homelab/vps-docker"}, ""))

	wrongScopeToken := s.CreateConsumerToken("unrelated-consumer")

	getCfg := cli.Config{Server: srv.URL, ConsumerToken: wrongScopeToken}
	_, err = cli.Get(context.Background(), getCfg, "x", []string{identity.String()})
	require.ErrorIs(t, err, client.ErrNotFound)
}

func TestGetFailsWithNoCredentialConfiguredAtAll(t *testing.T) {
	t.Parallel()

	srv, _, _ := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	getCfg := cli.Config{Server: srv.URL}
	_, err = cli.Get(context.Background(), getCfg, "x", []string{identity.String()})
	require.ErrorIs(t, err, client.ErrUnauthorized)
	require.ErrorContains(t, err, "--token")
	require.ErrorContains(t, err, "--consumer-token")
}

func TestGetFailsWithRejectedWriteToken(t *testing.T) {
	t.Parallel()

	srv, _, _ := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	getCfg := cli.Config{Server: srv.URL, Token: "not-a-real-token"}
	_, err = cli.Get(context.Background(), getCfg, "x", []string{identity.String()})
	require.ErrorIs(t, err, client.ErrUnauthorized)
	require.ErrorContains(t, err, "write token rejected")
}

func TestGetFailsWithRejectedConsumerToken(t *testing.T) {
	t.Parallel()

	srv, _, _ := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	getCfg := cli.Config{Server: srv.URL, ConsumerToken: "not-a-real-token"}
	_, err = cli.Get(context.Background(), getCfg, "x", []string{identity.String()})
	require.ErrorIs(t, err, client.ErrUnauthorized)
	require.ErrorContains(t, err, "consumer token rejected")
}
