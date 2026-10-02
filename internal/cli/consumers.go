package cli

import (
	"context"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// ListConsumers returns every consumer directory entry whose name contains
// query ("" matches all).
func ListConsumers(ctx context.Context, cfg Config, query string) ([]client.Consumer, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return nil, err
	}

	consumers, err := cl.ListConsumers(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list consumers: %w", err)
	}

	return consumers, nil
}

// AddConsumer adds name to the directory with no secret referencing it yet.
func AddConsumer(ctx context.Context, cfg Config, name string) (client.Consumer, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return client.Consumer{}, err
	}

	consumer, err := cl.AddConsumer(ctx, name)
	if err != nil {
		return client.Consumer{}, fmt.Errorf("add consumer: %w", err)
	}

	return consumer, nil
}

// UpdateConsumer renames name and/or registers its age public key - nil
// leaves a field unchanged.
func UpdateConsumer(ctx context.Context, cfg Config, name string, newName, publicKey *string) (client.Consumer, error) {
	cl, err := cfg.newClient()
	if err != nil {
		return client.Consumer{}, err
	}

	consumer, err := cl.UpdateConsumer(ctx, name, newName, publicKey)
	if err != nil {
		return client.Consumer{}, fmt.Errorf("update consumer: %w", err)
	}

	return consumer, nil
}

// DeleteConsumer removes name from the directory and every object's
// used_by list; no object is deleted.
func DeleteConsumer(ctx context.Context, cfg Config, name string) error {
	cl, err := cfg.newClient()
	if err != nil {
		return err
	}

	if err := cl.DeleteConsumer(ctx, name); err != nil {
		return fmt.Errorf("delete consumer: %w", err)
	}

	return nil
}
