package client

import (
	"context"
	"errors"
	"time"

	hushhush "github.com/alrayyes/hush-hush-go/v4"
)

// ErrConsumerTokenNotFound is returned by RotateConsumerToken and
// PurgeConsumerToken for an unknown, already revoked, or already expired
// token id - mapError's own ErrNotFound carries the wrong noun ("object
// not found") for this call.
var ErrConsumerTokenNotFound = errors.New("consumer token not found")

// ErrConsumerTokenActive is returned by PurgeConsumerToken for a token
// that's neither revoked nor expired yet - RevokeConsumerToken has to run
// first, matching the SDK's own PurgeConsumerToken doc comment.
var ErrConsumerTokenActive = errors.New("consumer token is still active - revoke it first")

// ConsumerToken is one issued consumer read token's metadata - never its
// raw value, which by design exists nowhere to return once
// CreateConsumerToken or RotateConsumerToken's caller has already seen it.
type ConsumerToken struct {
	ID          string     `json:"id"`
	Consumer    string     `json:"consumer"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	Revoked     bool       `json:"revoked"`
}

// ConsumerTokenWithValue is CreateConsumerToken and RotateConsumerToken's
// return value - ConsumerToken's own metadata plus the raw token, shown
// here once and never again.
type ConsumerTokenWithValue struct {
	ConsumerToken
	Value string `json:"value"`
}

// CreateConsumerToken mints a new read token scoped to consumer, valid
// for ttl starting now. Requires a write token - minting a credential
// that grants read access is itself a write-path operation
// (alrayyes/hush-hush#467).
func (c *Client) CreateConsumerToken(ctx context.Context, consumer, description string, ttl time.Duration) (ConsumerTokenWithValue, error) {
	resp, err := c.sdk.CreateConsumerToken(ctx, hushhush.CreateConsumerTokenRequest{
		Consumer:    consumer,
		Description: description,
		TtlSeconds:  int64(ttl.Seconds()),
	})
	if err != nil {
		return ConsumerTokenWithValue{}, mapError(err)
	}

	return toConsumerTokenWithValue(resp), nil
}

// ListConsumerTokens returns every issued consumer token's metadata,
// never a raw value. Requires a write token.
func (c *Client) ListConsumerTokens(ctx context.Context) ([]ConsumerToken, error) {
	tokens, err := c.sdk.ListConsumerTokens(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	result := make([]ConsumerToken, len(tokens))
	for i, t := range tokens {
		result[i] = toConsumerToken(t)
	}

	return result, nil
}

// RotateConsumerToken replaces id's secret and expiry, keeping its
// consumer and description unchanged - the old secret stops
// authenticating immediately. Requires a write token.
func (c *Client) RotateConsumerToken(ctx context.Context, id string, ttl time.Duration) (ConsumerTokenWithValue, error) {
	resp, err := c.sdk.RotateConsumerToken(ctx, id, hushhush.RotateConsumerTokenRequest{TtlSeconds: int64(ttl.Seconds())})
	if err != nil {
		return ConsumerTokenWithValue{}, mapConsumerTokenIDError(err)
	}

	return toConsumerTokenWithValue(resp), nil
}

// RevokeConsumerToken invalidates id - revoking one that's already
// expired or doesn't exist isn't an error, matching the SDK's own
// RevokeConsumerToken doc comment. Requires a write token.
func (c *Client) RevokeConsumerToken(ctx context.Context, id string) error {
	if err := c.sdk.RevokeConsumerToken(ctx, id); err != nil {
		return mapError(err)
	}

	return nil
}

// PurgeConsumerToken permanently removes id, once it's already revoked or
// past its expiry - RevokeConsumerToken's soft-delete stays the only way
// to invalidate a still-active token. Requires a write token.
func (c *Client) PurgeConsumerToken(ctx context.Context, id string) error {
	if err := c.sdk.PurgeConsumerToken(ctx, id); err != nil {
		return mapConsumerTokenIDError(err)
	}

	return nil
}

func toConsumerToken(t hushhush.ConsumerTokenMetadata) ConsumerToken {
	return ConsumerToken{
		ID:          t.Id,
		Consumer:    t.Consumer,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
		ExpiresAt:   t.ExpiresAt,
		LastUsedAt:  t.LastUsedAt,
		Revoked:     t.Revoked,
	}
}

func toConsumerTokenWithValue(t *hushhush.ConsumerTokenWithValue) ConsumerTokenWithValue {
	return ConsumerTokenWithValue{
		ConsumerToken{
			ID:          t.Id,
			Consumer:    t.Consumer,
			Description: t.Description,
			CreatedAt:   t.CreatedAt,
			ExpiresAt:   t.ExpiresAt,
			LastUsedAt:  t.LastUsedAt,
			Revoked:     t.Revoked,
		},
		t.Value,
	}
}

// mapConsumerTokenIDError translates mapError's generic, object-flavored
// sentinels into ones with the right noun for a token id - RotateConsumerToken
// and PurgeConsumerToken are the only calls id-scoped enough to need it.
func mapConsumerTokenIDError(err error) error {
	mapped := mapError(err)

	switch {
	case errors.Is(mapped, ErrNotFound):
		return ErrConsumerTokenNotFound
	case errors.Is(mapped, ErrAlreadyExists):
		return ErrConsumerTokenActive
	default:
		return mapped
	}
}
