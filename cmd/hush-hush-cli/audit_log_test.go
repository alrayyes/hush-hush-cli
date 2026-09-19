package main

import (
	"bytes"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/testserver"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLogHelpListsEverySevenFlag(t *testing.T) {
	cmd := newAuditLogCmd()

	for _, name := range []string{"object", "actor", "caller", "since", "until", "format", "limit"} {
		assert.NotNil(t, cmd.Flags().Lookup(name), "expected a --%s flag", name)
	}
}

func TestAuditLogRejectsAnUnknownFormatBeforeAnyRequest(t *testing.T) {
	t.Setenv("HUSH_HUSH_SERVER", "http://127.0.0.1:1") // unreachable - a request here would fail differently
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"audit-log", "--format", "xml"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "table or json")
}

func TestAuditLogRejectsAMalformedSinceBeforeAnyRequest(t *testing.T) {
	t.Setenv("HUSH_HUSH_SERVER", "http://127.0.0.1:1") // unreachable - a request here would fail differently
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	root.SetArgs([]string{"audit-log", "--since", "not-a-date"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--since")
}

func TestAuditLogNoFiltersReturnsEverything(t *testing.T) {
	srv, store, _ := testserver.New(t)

	store.RecordAuditEntry(testserver.AuditLogEntry{ObjectID: "secret-1", Action: "read", IP: "10.0.0.1"})
	store.RecordAuditEntry(testserver.AuditLogEntry{ObjectID: "secret-2", Action: "create", IP: "10.0.0.2"})

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"audit-log", "--format", "json"})

	require.NoError(t, root.Execute())

	var got []struct {
		ObjectID string `json:"object_id"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got, 2)
}

func TestAuditLogLimitBelowTheMatchingCountPrintsOnlyThatMany(t *testing.T) {
	srv, store, _ := testserver.New(t)

	for range 5 {
		store.RecordAuditEntry(testserver.AuditLogEntry{ObjectID: "secret-1", Action: "read", IP: "10.0.0.1"})
	}

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"audit-log", "--format", "json", "--limit", "3"})

	require.NoError(t, root.Execute())

	var got []struct{}
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got, 3)
}

func TestAuditLogLimitAtOrAboveTheMatchingCountPrintsEveryEntry(t *testing.T) {
	srv, store, _ := testserver.New(t)

	for range 3 {
		store.RecordAuditEntry(testserver.AuditLogEntry{ObjectID: "secret-1", Action: "read", IP: "10.0.0.1"})
	}

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"audit-log", "--format", "json", "--limit", "10"})

	require.NoError(t, root.Execute())

	var got []struct{}
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got, 3)
}

func TestAuditLogTableFormatShowsADashForANilCaller(t *testing.T) {
	srv, store, _ := testserver.New(t)

	store.RecordAuditEntry(testserver.AuditLogEntry{ObjectID: "secret-1", Action: "read", IP: "10.0.0.1"})

	t.Setenv("HUSH_HUSH_SERVER", srv.URL)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	viper.Reset()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"audit-log"})

	require.NoError(t, root.Execute())

	// The CALLER column (between OBJECT ID and IP) should be a bare "-"
	// placeholder for a nil caller, not blank or "<nil>".
	row := regexp.MustCompile(`read\s+secret-1\s+-\s+10\.0\.0\.1`)
	assert.Regexp(t, row, out.String())
}
