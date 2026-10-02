package client

import (
	"context"
	"errors"

	hushhush "github.com/alrayyes/hush-hush-go/v4"
)

// consumersPageMax is the server's own page-size cap (api/openapi.yaml's
// GET /consumers page_size parameter, max 100).
const consumersPageMax = 100

// ErrConsumerNotFound is returned by UpdateConsumer and DeleteConsumer for
// a name not in the directory - mapError's own ErrNotFound carries the
// wrong noun ("object not found") for these calls.
var ErrConsumerNotFound = errors.New("consumer not found")

// ErrConsumerAlreadyExists is returned by AddConsumer for a name already
// in the directory.
var ErrConsumerAlreadyExists = errors.New("consumer already exists")

// Consumer is one consumer directory entry. Matches
// components.schemas.ConsumerEntry in api/openapi.yaml.
type Consumer struct {
	Name        string `json:"name"`
	SecretCount int32  `json:"secret_count"`
	PublicKey   string `json:"public_key,omitempty"`
}

// ListConsumers returns every directory entry whose name contains query
// (case-insensitive; "" matches all), following the server's pagination so
// the caller always gets the whole set. Requires a write token.
func (c *Client) ListConsumers(ctx context.Context, query string) ([]Consumer, error) {
	var (
		all      []Consumer
		pageSize = int32(consumersPageMax)
	)

	for page := int32(1); ; page++ {
		filter := hushhush.ConsumerFilter{Page: &page, PageSize: &pageSize}
		if query != "" {
			filter.Q = &query
		}

		res, err := c.sdk.ListConsumers(ctx, filter)
		if err != nil {
			return nil, mapConsumerError(err)
		}

		if res.Page == nil {
			return all, nil
		}

		for _, e := range res.Page.Consumers {
			all = append(all, toConsumer(e))
		}

		if len(res.Page.Consumers) == 0 || len(all) >= int(res.Page.Total) {
			return all, nil
		}
	}
}

// AddConsumer adds name to the directory with no secret referencing it
// yet. Requires a write token.
func (c *Client) AddConsumer(ctx context.Context, name string) (Consumer, error) {
	e, err := c.sdk.AddConsumer(ctx, hushhush.AddConsumerRequest{Name: name})
	if err != nil {
		return Consumer{}, mapConsumerError(err)
	}

	return toConsumer(*e), nil
}

// UpdateConsumer renames name and/or registers its age public key - nil
// leaves a field unchanged. Requires a write token.
func (c *Client) UpdateConsumer(ctx context.Context, name string, newName, publicKey *string) (Consumer, error) {
	e, err := c.sdk.UpdateConsumer(ctx, name, hushhush.UpdateConsumerRequest{Name: newName, PublicKey: publicKey})
	if err != nil {
		return Consumer{}, mapConsumerError(err)
	}

	return toConsumer(*e), nil
}

// DeleteConsumer strips name from every object's used_by list and the
// directory; no object is deleted. Requires a write token.
func (c *Client) DeleteConsumer(ctx context.Context, name string) error {
	if err := c.sdk.DeleteConsumer(ctx, name); err != nil {
		return mapConsumerError(err)
	}

	return nil
}

func toConsumer(e hushhush.ConsumerEntry) Consumer {
	out := Consumer{Name: e.Name, SecretCount: e.SecretCount}
	if e.PublicKey != nil {
		out.PublicKey = *e.PublicKey
	}

	return out
}

// mapConsumerError swaps mapError's object-flavored sentinels for ones
// with the right noun for a consumer name.
func mapConsumerError(err error) error {
	mapped := mapError(err)

	switch {
	case errors.Is(mapped, ErrNotFound):
		return ErrConsumerNotFound
	case errors.Is(mapped, ErrAlreadyExists):
		return ErrConsumerAlreadyExists
	default:
		return mapped
	}
}
