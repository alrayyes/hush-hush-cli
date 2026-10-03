package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/seal"
)

// Update seals value to recipients and replaces id's stored value. Its
// used_by metadata stays unchanged (design.md) unless WithUsedBy replaces
// it, and WithTags replaces its tags - the writer's process never handles
// a private key here either, same as Inject.
//
// recipients, given non-empty, is used exactly as given. Left empty, the
// new consumers' registered public keys are resolved the way Inject does,
// and the call fails before touching the server's copy if one has none.
func Update(ctx context.Context, cfg Config, id string, value []byte, recipients []string, opts ...WriteOption) error {
	w := newWriteOptions(opts)

	cl, err := cfg.newClient()
	if err != nil {
		return err
	}

	var usedBy []string
	if w.usedBy != nil {
		usedBy = *w.usedBy
	}

	recipients, err = resolveRecipients(ctx, cl, recipients, usedBy)
	if err != nil {
		return err
	}

	sealed, err := seal.Seal(value, recipients)
	if err != nil {
		return fmt.Errorf("seal value: %w", err)
	}

	if _, err := cl.Update(ctx, id, sealed, w.tags, w.usedBy); err != nil {
		return fmt.Errorf("update object: %w", err)
	}

	return nil
}
