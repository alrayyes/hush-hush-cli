package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// AuditLog returns every audit log entry matching filter, oldest first -
// paging and limit handling both live in internal/client (design.md).
func AuditLog(ctx context.Context, cfg Config, filter client.AuditLogFilter) ([]client.AuditLogEntry, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return nil, err
	}

	entries, err := cl.QueryAuditLog(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("query audit log: %w", err)
	}

	return entries, nil
}
