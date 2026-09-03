//go:build integration

// Package integration boots the real, published hush-hush server image and
// proves internal/client's actual HTTP calls work against it - the layer
// none of internal/cli's or internal/client's own tests reach, since those
// exercise internal/testserver's hand-written fake rather than the real
// server binary someone actually deploys (rules/go-test.md: "The
// container/integration layer uses testcontainers-go"). Gated behind a
// build tag and its own CI job, the same shape hush-hush's own
// integration/container_test.go uses for its server-side equivalent -
// adapted here to pull the published ghcr.io/alrayyes/hush-hush image
// rather than building from a Dockerfile this repo doesn't own.
package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// containerServer and containerToken are set once by TestMain - the image
// pull and startup cost is paid once per run, shared by every test in this
// package, rather than each paying it separately.
var (
	containerServer string
	containerToken  string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "ghcr.io/alrayyes/hush-hush:latest",
			ExposedPorts: []string{"8080/tcp"},
			// A real file under the image's own baked-in, already-writable
			// /data, not :memory: - the token issued below runs as a
			// separate exec into this same container, which would get its
			// own, disconnected in-memory database otherwise.
			Env: map[string]string{
				"DB_PATH": "/data/hush-hush.db",
			},
			WaitingFor: wait.ForHTTP("/healthz").WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	containerServer, err = container.Endpoint(ctx, "http")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	execCode, reader, err := container.Exec(ctx,
		[]string{"/hush-hush", "token", "issue", "--description", "hush-hush-cli integration test"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "issue token:", err)
		os.Exit(1)
	}

	issueOut, err := io.ReadAll(reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read token issue output:", err)
		os.Exit(1)
	}

	if execCode != 0 {
		fmt.Fprintln(os.Stderr, "token issue exited nonzero:", string(issueOut))
		os.Exit(1)
	}

	var ok bool
	containerToken, ok = parseIssuedToken(string(issueOut))
	if !ok {
		fmt.Fprintln(os.Stderr, "could not find issued token in output:", string(issueOut))
		os.Exit(1)
	}

	exitCode := m.Run()

	if err := container.Terminate(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	os.Exit(exitCode)
}

// parseIssuedToken pulls the plaintext out of `hush-hush token issue`'s
// "token: <value>" line - the only machine-readable thing about that
// output, which is meant for a human running the command by hand.
func parseIssuedToken(output string) (token string, ok bool) {
	for _, line := range strings.Split(output, "\n") {
		if after, found := strings.CutPrefix(line, "token: "); found {
			return after, true
		}
	}

	return "", false
}

// TestContainerInjectGetUpdateDeleteRoundTrip drives internal/cli's actual
// functions - the same call chain the built binary makes - through the
// real hush-hush-go SDK against the real server, proving the whole client
// stack (age sealing, HTTP transport, error mapping) rather than just the
// server's own contract the way hush-hush's equivalent test does.
func TestContainerInjectGetUpdateDeleteRoundTrip(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	writeCfg := cli.Config{Server: containerServer, Token: containerToken}
	readCfg := cli.Config{Server: containerServer}

	value := []byte("container-integration-test-value")
	require.NoError(t, cli.Inject(t.Context(), writeCfg, "hush_hush_cli_integration_test", value,
		[]string{identity.Recipient().String()}, nil, "container integration test"))

	plaintext, err := cli.Get(t.Context(), readCfg, "hush_hush_cli_integration_test", []string{identity.String()})
	require.NoError(t, err)
	require.Equal(t, value, plaintext)

	newValue := []byte("updated-container-integration-test-value")
	require.NoError(t, cli.Update(t.Context(), writeCfg, "hush_hush_cli_integration_test", newValue,
		[]string{identity.Recipient().String()}))

	updated, err := cli.Get(t.Context(), readCfg, "hush_hush_cli_integration_test", []string{identity.String()})
	require.NoError(t, err)
	require.Equal(t, newValue, updated)

	require.NoError(t, cli.Delete(t.Context(), writeCfg, "hush_hush_cli_integration_test"))

	_, err = cli.Get(t.Context(), readCfg, "hush_hush_cli_integration_test", []string{identity.String()})
	require.ErrorIs(t, err, client.ErrNotFound)
}

// TestContainerRejectsBadToken proves the real server's 401 semantics map
// through internal/client the same way internal/testserver's fake already
// does - the fake's fidelity is exactly the thing this package exists to
// check.
func TestContainerRejectsBadToken(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	cfg := cli.Config{Server: containerServer, Token: "wrong-token"}

	err = cli.Inject(t.Context(), cfg, "hush_hush_cli_integration_test_bad_token", []byte("v"),
		[]string{identity.Recipient().String()}, nil, "")
	require.ErrorIs(t, err, client.ErrUnauthorized)
}
