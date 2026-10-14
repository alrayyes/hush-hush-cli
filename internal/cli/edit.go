package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// ErrNothingToEdit is Edit called with no field to change, which the server
// would only refuse with a 400 of its own.
var ErrNothingToEdit = errors.New("nothing to edit: give a description, tags or a slug")

// Edit changes id's description, tags or name (WithDescription, WithTags,
// WithSlug) without reading or sealing a value, so fixing a typo needs no
// plaintext. A field left out stays as it is.
func Edit(ctx context.Context, cfg Config, id string, opts ...WriteOption) error {
	w := newWriteOptions(opts)

	if w.description == nil && w.slug == nil && w.tags == nil {
		return ErrNothingToEdit
	}

	cl, err := cfg.newClient()
	if err != nil {
		return err
	}

	if _, err := cl.Patch(ctx, id, w.description, w.slug, w.tags, client.WithVariant(w.variantID), client.WithConsumer(w.consumer)); err != nil {
		return fmt.Errorf("edit object: %w", err)
	}

	return nil
}
