package cli_test

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/require"
)

// newTestServer issues a fresh write token valid against its own store -
// none of this package's tests need a distinct one.
func newTestServer(t *testing.T) (srv *httptest.Server, s *testserver.Store, token string) {
	t.Helper()

	return testserver.New(t)
}

func TestInjectCreatesAnObjectTheMatchingIdentityCanDecrypt(t *testing.T) {
	t.Parallel()

	srv, s, token := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	cfg := cli.Config{Server: srv.URL, Token: token}
	value := []byte("plaintext-value")

	err = cli.Inject(context.Background(), cfg, "mattermost_deploy_webhook", value,
		[]string{identity.Recipient().String()}, []string{"homelab/vps-docker"}, "prod deploy webhook")
	require.NoError(t, err)

	obj, err := s.GetObject(context.Background(), "mattermost_deploy_webhook")
	require.NoError(t, err)
	require.Equal(t, []string{"homelab/vps-docker"}, obj.UsedBy)
	require.Equal(t, "prod deploy webhook", obj.Description)
	require.NotEqual(t, value, obj.Value, "stored value must be sealed, not plaintext")

	r, err := age.Decrypt(bytes.NewReader(obj.Value), identity)
	require.NoError(t, err)

	plaintext, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, value, plaintext)
}

func TestInjectWithoutAValidTokenIsRejected(t *testing.T) {
	t.Parallel()

	srv, _, _ := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	cfg := cli.Config{Server: srv.URL, Token: "wrong-token"}

	err = cli.Inject(context.Background(), cfg, "x", []byte("v"), []string{identity.Recipient().String()}, nil, "")
	require.ErrorIs(t, err, client.ErrUnauthorized)
}

func TestInjectWithNoRecipientsResolvesUsedByConsumersRegisteredKeys(t *testing.T) {
	t.Parallel()

	srv, s, token := newTestServer(t)

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	require.NoError(t, s.AddConsumer(context.Background(), "homelab/vps-docker"))
	recipient := identity.Recipient().String()
	_, err = s.UpdateConsumer(context.Background(), "homelab/vps-docker", nil, &recipient)
	require.NoError(t, err)

	cfg := cli.Config{Server: srv.URL, Token: token}
	value := []byte("plaintext-value")

	err = cli.Inject(context.Background(), cfg, "mattermost_deploy_webhook", value,
		nil, []string{"homelab/vps-docker"}, "")
	require.NoError(t, err)

	obj, err := s.GetObject(context.Background(), "mattermost_deploy_webhook")
	require.NoError(t, err)

	r, err := age.Decrypt(bytes.NewReader(obj.Value), identity)
	require.NoError(t, err)

	plaintext, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, value, plaintext)
}

func TestInjectWithExplicitRecipientsOverridesUsedByResolution(t *testing.T) {
	t.Parallel()

	srv, s, token := newTestServer(t)

	registered, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	explicit, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	require.NoError(t, s.AddConsumer(context.Background(), "homelab/vps-docker"))
	registeredRecipient := registered.Recipient().String()
	_, err = s.UpdateConsumer(context.Background(), "homelab/vps-docker", nil, &registeredRecipient)
	require.NoError(t, err)

	cfg := cli.Config{Server: srv.URL, Token: token}
	value := []byte("plaintext-value")

	err = cli.Inject(context.Background(), cfg, "mattermost_deploy_webhook", value,
		[]string{explicit.Recipient().String()}, []string{"homelab/vps-docker"}, "")
	require.NoError(t, err)

	obj, err := s.GetObject(context.Background(), "mattermost_deploy_webhook")
	require.NoError(t, err)

	_, err = age.Decrypt(bytes.NewReader(obj.Value), registered)
	require.Error(t, err, "registered consumer key must not be a recipient once --recipients is explicit")

	r, err := age.Decrypt(bytes.NewReader(obj.Value), explicit)
	require.NoError(t, err)

	plaintext, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, value, plaintext)
}

func TestInjectWithAUsedByConsumerWithNoRegisteredKeyFailsNamingIt(t *testing.T) {
	t.Parallel()

	srv, s, token := newTestServer(t)

	require.NoError(t, s.AddConsumer(context.Background(), "homelab/vps-docker"))

	cfg := cli.Config{Server: srv.URL, Token: token}

	err := cli.Inject(context.Background(), cfg, "mattermost_deploy_webhook", []byte("v"),
		nil, []string{"homelab/vps-docker"}, "")
	require.ErrorIs(t, err, client.ErrConsumerNoPublicKey)
	require.ErrorContains(t, err, "homelab/vps-docker")

	_, getErr := s.GetObject(context.Background(), "mattermost_deploy_webhook")
	require.ErrorIs(t, getErr, testserver.ErrNotFound, "a resolution failure must not reach the server")
}

func TestInjectWithAMalformedRecipientFailsBeforeCallingTheServer(t *testing.T) {
	t.Parallel()

	srv, s, token := newTestServer(t)

	cfg := cli.Config{Server: srv.URL, Token: token}

	err := cli.Inject(context.Background(), cfg, "x", []byte("v"), []string{"not-a-recipient"}, nil, "")
	require.Error(t, err)

	_, getErr := s.GetObject(context.Background(), "x")
	require.ErrorIs(t, getErr, testserver.ErrNotFound, "a sealing failure must not reach the server")
}
