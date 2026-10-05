package render

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// OneConsumer prints a single consumer as a one-row table, or raw JSON.
func OneConsumer(out io.Writer, consumer client.Consumer, asJSON bool) error {
	if asJSON {
		if err := json.NewEncoder(out).Encode(consumer); err != nil {
			return fmt.Errorf("write consumer as json: %w", err)
		}

		return nil
	}

	return ConsumersTable(out, []client.Consumer{consumer})
}

// ConsumersJSON prints consumers as one JSON array.
func ConsumersJSON(out io.Writer, consumers []client.Consumer) error {
	if err := json.NewEncoder(out).Encode(consumers); err != nil {
		return fmt.Errorf("write consumers as json: %w", err)
	}

	return nil
}

// ConsumersTable prints consumers as an aligned table.
func ConsumersTable(out io.Writer, consumers []client.Consumer) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintln(w, "NAME\tSECRETS\tPUBLIC KEY"); err != nil {
		return fmt.Errorf("write consumer header: %w", err)
	}

	for _, c := range consumers {
		if _, err := fmt.Fprintf(w, "%s\t%d\t%s\n", c.Name, c.SecretCount, c.PublicKey); err != nil {
			return fmt.Errorf("write consumer row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush consumer table: %w", err)
	}

	return nil
}
