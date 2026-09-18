package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// List returns every stored object's metadata - there's nothing to seal or
// unseal here, unlike get/inject/update.
func List(ctx context.Context, cfg Config) ([]client.ObjectMetadata, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return nil, err
	}

	objects, err := cl.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}

	return objects, nil
}
