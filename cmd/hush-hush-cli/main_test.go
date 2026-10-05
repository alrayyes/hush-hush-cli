//go:build unix

package main_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInterruptCancelsARequestToAServerThatNeverAnswers runs the real
// binary: signal handling lives in main, so nothing short of a process
// and a signal exercises it.
func TestInterruptCancelsARequestToAServerThatNeverAnswers(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "hush-hush-cli")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".") //nolint:gosec // fixed arguments, a path under t.TempDir()
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))

	release := make(chan struct{})
	hanging := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-release
	}))

	t.Cleanup(hanging.Close)
	t.Cleanup(func() { close(release) })

	var stderr bytes.Buffer

	cmd := exec.CommandContext(t.Context(), bin, "list") //nolint:gosec // the binary this test just built
	cmd.Env = append(os.Environ(),
		"HUSH_HUSH_SERVER="+hanging.URL,
		"HUSH_HUSH_TOKEN=token",
		"XDG_CONFIG_HOME="+t.TempDir(),
	)
	cmd.Stderr = &stderr

	require.NoError(t, cmd.Start())

	// Give it time to reach the request and block on the server.
	time.Sleep(500 * time.Millisecond)
	require.NoError(t, cmd.Process.Signal(syscall.SIGINT))

	done := make(chan error, 1)

	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		var exitErr *exec.ExitError

		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 130, exitErr.ExitCode(), "the shell convention for a run ended by SIGINT")
		assert.Contains(t, stderr.String(), "interrupted")
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()

		t.Fatal("the CLI kept running after SIGINT")
	}
}
