package cli_test

import (
	"testing"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAuditLogFilterLeavesUnsetFieldsNil(t *testing.T) {
	t.Parallel()

	filter, err := cli.ParseAuditLogFilter("", "", "", "", "", 0)

	require.NoError(t, err)
	assert.Nil(t, filter.ObjectID)
	assert.Nil(t, filter.Token)
	assert.Nil(t, filter.Caller)
	assert.Nil(t, filter.Since)
	assert.Nil(t, filter.Until)
	assert.Nil(t, filter.Limit)
}

func TestParseAuditLogFilterSetsEveryGivenField(t *testing.T) {
	t.Parallel()

	filter, err := cli.ParseAuditLogFilter("obj", "tok", "ci", "2026-01-02T03:04:05Z", "2026-02-03T04:05:06Z", 7)

	require.NoError(t, err)
	assert.Equal(t, "obj", *filter.ObjectID)
	assert.Equal(t, "tok", *filter.Token, "--actor maps onto the server's verified actor filter")
	assert.Equal(t, "ci", *filter.Caller)
	assert.True(t, filter.Since.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)))
	assert.True(t, filter.Until.Equal(time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)))
	assert.Equal(t, 7, *filter.Limit)
}

func TestParseAuditLogFilterNamesTheFlagThatFailedToParse(t *testing.T) {
	t.Parallel()

	_, err := cli.ParseAuditLogFilter("", "", "", "not-a-date", "", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--since")

	_, err = cli.ParseAuditLogFilter("", "", "", "", "yesterday", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--until")
}

func TestParseTTL(t *testing.T) {
	t.Parallel()

	got, err := cli.ParseTTL("720h")
	require.NoError(t, err)
	assert.Equal(t, 720*time.Hour, got)

	_, err = cli.ParseTTL("")
	require.ErrorIs(t, err, cli.ErrTTLRequired)

	_, err = cli.ParseTTL("a month")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--ttl")
}
