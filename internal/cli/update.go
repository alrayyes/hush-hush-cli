package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/seal"
)

// Update seals value to recipients and replaces id's stored value, leaving
// its used_by metadata unchanged (design.md; WithTags replaces its tags) - the writer's process never
// handles a private key here either, same as Inject.
func Update(ctx context.Context, cfg Config, id string, value []byte, recipients []string, opts ...WriteOption) error {
	w := newWriteOptions(opts)

	sealed, err := seal.Seal(value, recipients)
	if err != nil {
		return fmt.Errorf("seal value: %w", err)
	}

	cl, err := cfg.newClient()
	if err != nil {
		return err
	}

	if _, err := cl.Update(ctx, id, sealed, w.tags); err != nil {
		return fmt.Errorf("update object: %w", err)
	}

	return nil
}
