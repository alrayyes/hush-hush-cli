package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// WriteOption tweaks an Inject or Update call beyond its required
// arguments - variadic so adding one never touches existing callers.
type WriteOption func(*writeOptions)

type writeOptions struct {
	// tags is nil when unset. On Update a non-nil empty slice clears the
	// object's tags; on Inject empty and nil are the same.
	tags *[]string
	// usedBy is nil when unset. On Update a non-nil empty slice clears the
	// object's consumers.
	usedBy *[]string
	// keepReadableCopy adds the owner's escrowed key as a recipient.
	keepReadableCopy bool
}

func newWriteOptions(opts []WriteOption) writeOptions {
	var w writeOptions
	for _, o := range opts {
		o(&w)
	}

	return w
}

// tagsOrNil is the tags to send on create: nothing, unless some were given.
func (w writeOptions) tagsOrNil() []string {
	if w.tags == nil {
		return nil
	}

	return *w.tags
}

// WithTags sets an object's tags: on Inject the initial ones, on Update a
// full replacement. Pass an empty, non-nil slice to Update to clear them.
func WithTags(tags []string) WriteOption {
	return func(w *writeOptions) { w.tags = &tags }
}

// WithUsedBy replaces an object's consumers on Update. Pass an empty,
// non-nil slice to clear them. Inject takes its consumers as an argument.
func WithUsedBy(usedBy []string) WriteOption {
	return func(w *writeOptions) { w.usedBy = &usedBy }
}

// WithKeepReadableCopy seals the value to the owner's escrowed identity
// key as well, so the owner can decrypt what they wrote. The server never
// does this itself.
func WithKeepReadableCopy() WriteOption {
	return func(w *writeOptions) { w.keepReadableCopy = true }
}

// withOwnerKey appends the owner's escrowed public key to recipients when
// keepReadableCopy is set. It fails, rather than sealing without it, when
// the owner has none.
func (w writeOptions) withOwnerKey(ctx context.Context, cl *client.Client, recipients []string) ([]string, error) {
	if !w.keepReadableCopy {
		return recipients, nil
	}

	key, err := cl.OwnerPublicKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("keep readable copy: %w", err)
	}

	return append(recipients, key), nil
}
