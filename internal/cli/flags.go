package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// ParseAuditLogFilter parses --since/--until as RFC3339 (design.md) before
// any request is made, so a malformed value fails locally rather than as
// a 400 from the server.
func ParseAuditLogFilter(objectID, actor, caller, since, until string, limit int) (client.AuditLogFilter, error) {
	var filter client.AuditLogFilter

	if objectID != "" {
		filter.ObjectID = &objectID
	}

	if actor != "" {
		filter.Token = &actor
	}

	if caller != "" {
		filter.Caller = &caller
	}

	if since != "" {
		t, err := time.Parse(time.RFC3339, since)
		if err != nil {
			return filter, fmt.Errorf("--since: %w", err)
		}

		filter.Since = &t
	}

	if until != "" {
		t, err := time.Parse(time.RFC3339, until)
		if err != nil {
			return filter, fmt.Errorf("--until: %w", err)
		}

		filter.Until = &t
	}

	if limit > 0 {
		filter.Limit = &limit
	}

	return filter, nil
}

// ErrTTLRequired is a sentinel: a fixed condition (no --ttl given), not a
// message built from per-call detail.
var ErrTTLRequired = errors.New("--ttl is required (a Go duration, e.g. 720h)")

// ParseTTL reads --ttl as a Go duration (720h, say). It's required: a
// consumer token with no expiry isn't offered.
func ParseTTL(ttl string) (time.Duration, error) {
	if ttl == "" {
		return 0, ErrTTLRequired
	}

	d, err := time.ParseDuration(ttl)
	if err != nil {
		return 0, fmt.Errorf("--ttl: %w", err)
	}

	return d, nil
}
