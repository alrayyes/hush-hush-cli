package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// List returns stored objects' metadata - only those whose used_by
// includes usedBy, when it's non-empty. There's nothing to seal or unseal
// here, unlike get/inject/update.
func List(ctx context.Context, cfg Config, usedBy string) ([]client.ObjectMetadata, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return nil, err
	}

	objects, err := cl.List(ctx, usedBy)
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}

	return objects, nil
}
