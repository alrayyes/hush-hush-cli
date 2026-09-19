package cli_test

import (
	"context"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/require"
)

func TestAuditLogPassesTheGivenFilterThrough(t *testing.T) {
	t.Parallel()

	srv, store, _ := newTestServer(t)

	objectID := "secret-1"
	store.RecordAuditEntry(testserver.AuditLogEntry{ObjectID: objectID, Action: "read", IP: "10.0.0.1"})
	store.RecordAuditEntry(testserver.AuditLogEntry{ObjectID: "secret-2", Action: "read", IP: "10.0.0.1"})

	cfg := cli.Config{Server: srv.URL}
	entries, err := cli.AuditLog(context.Background(), cfg, client.AuditLogFilter{ObjectID: &objectID})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, objectID, entries[0].ObjectID)
}

func TestAuditLogEmptyLogReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	srv, _, _ := newTestServer(t)

	cfg := cli.Config{Server: srv.URL}
	entries, err := cli.AuditLog(context.Background(), cfg, client.AuditLogFilter{})
	require.NoError(t, err)
	require.Empty(t, entries)
}
