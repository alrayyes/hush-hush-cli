package client_test

import (
	"testing"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func TestQueryAuditLogFiltersCombineWithAND(t *testing.T) {
	t.Parallel()

	srv, store, _ := testserver.New(t)

	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	// Matches every filter below.
	store.RecordAuditEntry(testserver.AuditLogEntry{
		ObjectID: "secret-1", Action: "read", Caller: ptr("ci-runner"),
		ActorID: ptr("tok_abc"), IP: "10.0.0.1", Timestamp: base.Add(time.Hour),
	})
	// Wrong object.
	store.RecordAuditEntry(testserver.AuditLogEntry{
		ObjectID: "secret-2", Action: "read", Caller: ptr("ci-runner"),
		ActorID: ptr("tok_abc"), IP: "10.0.0.1", Timestamp: base.Add(time.Hour),
	})
	// Wrong caller.
	store.RecordAuditEntry(testserver.AuditLogEntry{
		ObjectID: "secret-1", Action: "read", Caller: ptr("other-caller"),
		ActorID: ptr("tok_abc"), IP: "10.0.0.1", Timestamp: base.Add(time.Hour),
	})
	// Wrong actor.
	store.RecordAuditEntry(testserver.AuditLogEntry{
		ObjectID: "secret-1", Action: "read", Caller: ptr("ci-runner"),
		ActorID: ptr("tok_xyz"), IP: "10.0.0.1", Timestamp: base.Add(time.Hour),
	})
	// Outside the since/until window.
	store.RecordAuditEntry(testserver.AuditLogEntry{
		ObjectID: "secret-1", Action: "read", Caller: ptr("ci-runner"),
		ActorID: ptr("tok_abc"), IP: "10.0.0.1", Timestamp: base.Add(-time.Hour),
	})

	cl, err := client.New(srv.URL, "")
	require.NoError(t, err)

	entries, err := cl.QueryAuditLog(t.Context(), client.AuditLogFilter{
		ObjectID: ptr("secret-1"),
		Caller:   ptr("ci-runner"),
		Token:    ptr("tok_abc"),
		Since:    ptr(base),
		Until:    ptr(base.Add(2 * time.Hour)),
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "secret-1", entries[0].ObjectID)
}

func TestQueryAuditLogSinglePageReturnsAllMatchingEntries(t *testing.T) {
	t.Parallel()

	srv, store, _ := testserver.New(t)

	for i := range 10 {
		store.RecordAuditEntry(testserver.AuditLogEntry{
			ObjectID: "secret-1", Action: "read", IP: "10.0.0.1",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	cl, err := client.New(srv.URL, "")
	require.NoError(t, err)

	entries, err := cl.QueryAuditLog(t.Context(), client.AuditLogFilter{})
	require.NoError(t, err)
	require.Len(t, entries, 10)
}

func TestQueryAuditLogPagesThroughEveryEntryInOrder(t *testing.T) {
	t.Parallel()

	srv, store, _ := testserver.New(t)

	const total = 700 // more than the server's 500-per-page cap

	for i := range total {
		store.RecordAuditEntry(testserver.AuditLogEntry{
			ObjectID: "secret-1", Action: "read", IP: "10.0.0.1",
			Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond),
		})
	}

	cl, err := client.New(srv.URL, "")
	require.NoError(t, err)

	entries, err := cl.QueryAuditLog(t.Context(), client.AuditLogFilter{})
	require.NoError(t, err)
	require.Len(t, entries, total)

	for i, e := range entries {
		require.Equal(t, int64(i+1), e.ID, "entries must come back oldest-first, in id order")
	}
}

func TestQueryAuditLogRespectsLimitAcrossPages(t *testing.T) {
	t.Parallel()

	srv, store, _ := testserver.New(t)

	for i := range 700 {
		store.RecordAuditEntry(testserver.AuditLogEntry{
			ObjectID: "secret-1", Action: "read", IP: "10.0.0.1",
			Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond),
		})
	}

	cl, err := client.New(srv.URL, "")
	require.NoError(t, err)

	entries, err := cl.QueryAuditLog(t.Context(), client.AuditLogFilter{Limit: ptr(600)})
	require.NoError(t, err)
	require.Len(t, entries, 600)
}

func TestQueryAuditLogUnexpectedStatusMapsToSentinel(t *testing.T) {
	t.Parallel()

	srv, _, _ := testserver.New(t)

	cl, err := client.New(srv.URL, "")
	require.NoError(t, err)

	_, err = cl.QueryAuditLog(t.Context(), client.AuditLogFilter{ObjectID: ptr("Not_Valid!")})
	require.ErrorIs(t, err, client.ErrUnexpectedStatus)
}
