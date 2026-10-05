package render_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	when    = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	whenStr = when.Local().Format(time.DateTime)
)

// lines splits a table into its non-empty lines with runs of spaces
// collapsed, so a test checks the cells and not the column padding.
func lines(out string) []string {
	var got []string

	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		got = append(got, strings.Join(strings.Fields(line), " "))
	}

	return got
}

func TestObjectsTableHasEveryColumnAndDashesForMissingTimes(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	require.NoError(t, render.ObjectsTable(&out, []client.ObjectMetadata{
		{Slug: "a", UsedBy: []string{"x", "y"}, Tags: []string{"t1"}, CreatedAt: &when, Description: "d"},
		{Slug: "b", UsedBy: []string{"u"}, Tags: []string{"t"}, Description: "d2"},
	}))

	assert.Equal(t, []string{
		"ID USED BY TAGS CREATED UPDATED DESCRIPTION",
		"a x,y t1 " + whenStr + " - d",
		"b u t - - d2",
	}, lines(out.String()))
}

func TestObjectsJSONPrintsAnEmptyArrayNotNull(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	require.NoError(t, render.ObjectsJSON(&out, nil))
	assert.Equal(t, "[]\n", out.String())
}

func TestConsumerTokenWithValueTableIncludesTheValueAndJSONIsRaw(t *testing.T) {
	t.Parallel()

	token := client.ConsumerTokenWithValue{Value: "secret-value"}
	token.ID, token.Consumer, token.Description, token.ExpiresAt = "id1", "c", "d", when

	var table, raw bytes.Buffer

	require.NoError(t, render.ConsumerTokenWithValue(&table, token, false))
	assert.Equal(t, []string{
		"ID CONSUMER DESCRIPTION EXPIRES TOKEN",
		"id1 c d " + whenStr + " secret-value",
	}, lines(table.String()))

	require.NoError(t, render.ConsumerTokenWithValue(&raw, token, true))
	assert.Contains(t, raw.String(), `"value":"secret-value"`)
}

func TestConsumerTokensTableShowsStatusAndJSONOfNilIsAnEmptyArray(t *testing.T) {
	t.Parallel()

	var table, raw bytes.Buffer

	require.NoError(t, render.ConsumerTokensTable(&table, []client.ConsumerToken{
		{ID: "id1", Consumer: "c", Description: "d", ExpiresAt: when, Status: "active"},
	}))
	assert.Equal(t, []string{
		"ID CONSUMER DESCRIPTION EXPIRES STATUS",
		"id1 c d " + whenStr + " active",
	}, lines(table.String()))

	require.NoError(t, render.ConsumerTokensJSON(&raw, nil))
	assert.Equal(t, "[]\n", raw.String())
}

func TestConsumersTableAndOneConsumer(t *testing.T) {
	t.Parallel()

	consumer := client.Consumer{Name: "c", SecretCount: 3, PublicKey: "age1abc"}

	var table, one, oneJSON bytes.Buffer

	require.NoError(t, render.ConsumersTable(&table, []client.Consumer{consumer}))
	assert.Equal(t, []string{"NAME SECRETS PUBLIC KEY", "c 3 age1abc"}, lines(table.String()))

	require.NoError(t, render.OneConsumer(&one, consumer, false))
	assert.Equal(t, lines(table.String()), lines(one.String()), "a single consumer prints as a one-row table")

	require.NoError(t, render.OneConsumer(&oneJSON, consumer, true))
	assert.JSONEq(t, `{"name":"c","secret_count":3,"public_key":"age1abc"}`, oneJSON.String())
}

func TestConsumersJSONIsAnArray(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	require.NoError(t, render.ConsumersJSON(&out, []client.Consumer{{Name: "c"}}))
	assert.JSONEq(t, `[{"name":"c","secret_count":0}]`, out.String())
}

func TestAuditLogTableShowsADashForANilCaller(t *testing.T) {
	t.Parallel()

	caller := "ci"

	var out bytes.Buffer

	require.NoError(t, render.AuditLogTable(&out, []client.AuditLogEntry{
		{Timestamp: when, Action: "read", ObjectID: "o1", Caller: &caller, IP: "10.0.0.1"},
		{Timestamp: when, Action: "create", ObjectID: "o2", IP: "10.0.0.2"},
	}))

	assert.Equal(t, []string{
		"TIMESTAMP ACTION OBJECT ID CALLER IP",
		whenStr + " read o1 ci 10.0.0.1",
		whenStr + " create o2 - 10.0.0.2",
	}, lines(out.String()))
}

func TestAuditLogJSONPrintsAnEmptyArrayNotNull(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	require.NoError(t, render.AuditLogJSON(&out, nil))
	assert.Equal(t, "[]\n", out.String())
}

func TestStatusJSON(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	require.NoError(t, render.StatusJSON(&out, client.AuthStatus{Bootstrapped: true}))
	assert.JSONEq(t, `{"bootstrapped":true}`, out.String())
}

func TestUsedByLinesAndJSON(t *testing.T) {
	t.Parallel()

	var lined, raw, empty bytes.Buffer

	require.NoError(t, render.UsedByLines(&lined, []string{"a", "b"}))
	assert.Equal(t, "a\nb\n", lined.String())

	require.NoError(t, render.UsedByJSON(&raw, []string{"a"}))
	assert.Equal(t, "[\"a\"]\n", raw.String())

	require.NoError(t, render.UsedByJSON(&empty, nil))
	assert.Equal(t, "[]\n", empty.String())
}
