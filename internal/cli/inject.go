package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/seal"
)

// Inject seals value to recipients and creates a new object under id - the
// writer's process never handles a private key, only recipients' public
// keys (design.md). description is fixed at creation, the same as usedBy.
//
// recipients, given non-empty, is used exactly as given - an explicit
// --recipients always wins over directory resolution. Left empty, each of
// usedBy's consumers' registered public key is resolved from the server's
// consumer directory instead (issue #125), and the call fails naming the
// first consumer with no registered key rather than sealing to fewer
// recipients than requested.
func Inject(ctx context.Context, cfg Config, id string, value []byte, recipients []string, usedBy []string, description string) error {
	cl, err := cfg.newClient()
	if err != nil {
		return err
	}

	recipients, err = resolveRecipients(ctx, cl, recipients, usedBy)
	if err != nil {
		return err
	}

	sealed, err := seal.Seal(value, recipients)
	if err != nil {
		return fmt.Errorf("seal value: %w", err)
	}

	if _, err := cl.Create(ctx, id, sealed, usedBy, description); err != nil {
		return fmt.Errorf("create object: %w", err)
	}

	return nil
}

// resolveRecipients returns recipients unchanged when non-empty. Otherwise
// it resolves each of usedBy's consumers' registered public key from the
// server's consumer directory.
func resolveRecipients(ctx context.Context, cl *client.Client, recipients, usedBy []string) ([]string, error) {
	if len(recipients) > 0 {
		return recipients, nil
	}

	resolved := make([]string, 0, len(usedBy))

	for _, name := range usedBy {
		key, err := cl.ConsumerPublicKey(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("resolve --used-by recipients: %w", err)
		}

		resolved = append(resolved, key)
	}

	return resolved, nil
}
