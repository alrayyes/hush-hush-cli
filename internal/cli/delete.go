package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// Delete permanently removes id - there's nothing to seal or unseal here,
// unlike every other command.
func Delete(ctx context.Context, cfg Config, id string, opts ...client.CallOption) error {
	cl, err := cfg.newClient()
	if err != nil {
		return err
	}

	if err := cl.Delete(ctx, id, opts...); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}

	return nil
}
