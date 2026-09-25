package cli

import (
	"context"
	"fmt"
)

// UsedBy returns id's recorded consumers - unauthenticated, matching Get,
// not List: enumerating one object's own already-known-id lineage grants
// no capability List's full enumeration does.
func UsedBy(ctx context.Context, cfg Config, id string) ([]string, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return nil, err
	}

	usedBy, err := cl.UsedBy(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("used-by: %w", err)
	}

	return usedBy, nil
}
