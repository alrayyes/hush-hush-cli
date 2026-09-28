package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// CreateConsumerToken mints a new consumer read token - the caller sets
// cfg.RequireToken, the same as inject/update/delete: minting a
// credential that grants read access is itself a write-path operation.
func CreateConsumerToken(ctx context.Context, cfg Config, consumer, description string, ttl time.Duration) (client.ConsumerTokenWithValue, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return client.ConsumerTokenWithValue{}, err
	}

	token, err := cl.CreateConsumerToken(ctx, consumer, description, ttl)
	if err != nil {
		return client.ConsumerTokenWithValue{}, fmt.Errorf("create consumer token: %w", err)
	}

	return token, nil
}

// ListConsumerTokens returns every issued consumer token's metadata,
// never a raw value.
func ListConsumerTokens(ctx context.Context, cfg Config) ([]client.ConsumerToken, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return nil, err
	}

	tokens, err := cl.ListConsumerTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("list consumer tokens: %w", err)
	}

	return tokens, nil
}

// RotateConsumerToken replaces id's secret and expiry, keeping its
// consumer and description unchanged.
func RotateConsumerToken(ctx context.Context, cfg Config, id string, ttl time.Duration) (client.ConsumerTokenWithValue, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return client.ConsumerTokenWithValue{}, err
	}

	token, err := cl.RotateConsumerToken(ctx, id, ttl)
	if err != nil {
		return client.ConsumerTokenWithValue{}, fmt.Errorf("rotate consumer token: %w", err)
	}

	return token, nil
}

// RevokeConsumerToken invalidates id - revoking one that's already
// expired or doesn't exist isn't an error.
func RevokeConsumerToken(ctx context.Context, cfg Config, id string) error {
	cl, err := cfg.newClient()
	if err != nil {
		return err
	}

	if err := cl.RevokeConsumerToken(ctx, id); err != nil {
		return fmt.Errorf("revoke consumer token: %w", err)
	}

	return nil
}

// PurgeConsumerToken permanently removes id, once it's already revoked or
// past its expiry.
func PurgeConsumerToken(ctx context.Context, cfg Config, id string) error {
	cl, err := cfg.newClient()
	if err != nil {
		return err
	}

	if err := cl.PurgeConsumerToken(ctx, id); err != nil {
		return fmt.Errorf("purge consumer token: %w", err)
	}

	return nil
}
