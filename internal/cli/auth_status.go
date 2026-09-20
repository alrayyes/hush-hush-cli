package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// AuthStatus reports whether the target server has an admin account
// bootstrapped yet - unauthenticated, unlike list/inject/update/delete.
func AuthStatus(ctx context.Context, cfg Config) (client.AuthStatus, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return client.AuthStatus{}, err
	}

	status, err := cl.AuthStatus(ctx)
	if err != nil {
		return client.AuthStatus{}, fmt.Errorf("auth status: %w", err)
	}

	return status, nil
}
