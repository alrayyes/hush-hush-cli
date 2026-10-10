package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// Health reports whether the target server is up - unauthenticated, like
// AuthStatus.
func Health(ctx context.Context, cfg Config) (client.Health, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return client.Health{}, err
	}

	health, err := cl.Health(ctx)
	if err != nil {
		return client.Health{}, fmt.Errorf("health: %w", err)
	}

	return health, nil
}
