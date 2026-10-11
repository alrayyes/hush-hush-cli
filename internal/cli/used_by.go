package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// UsedBy returns id's recorded consumers - unauthenticated, matching Get,
// not List: enumerating one object's own already-known-id lineage grants
// no capability List's full enumeration does.
func UsedBy(ctx context.Context, cfg Config, id string, opts ...client.CallOption) ([]string, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return nil, err
	}

	usedBy, err := cl.UsedBy(ctx, id, opts...)
	if err != nil {
		return nil, fmt.Errorf("used-by: %w", err)
	}

	return usedBy, nil
}
