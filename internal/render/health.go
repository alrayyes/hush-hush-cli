package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// HealthJSON prints the health answer as JSON.
func HealthJSON(out io.Writer, health client.Health) error {
	if err := json.NewEncoder(out).Encode(health); err != nil {
		return fmt.Errorf("write health as json: %w", err)
	}

	return nil
}

// HealthText prints the health answer as plain lines, leaving out an
// environment label the server doesn't set.
func HealthText(out io.Writer, health client.Health) error {
	if _, err := fmt.Fprintf(out, "status: %s\n", health.Status); err != nil {
		return fmt.Errorf("write health: %w", err)
	}

	if health.Environment != nil {
		if _, err := fmt.Fprintf(out, "environment: %s\n", *health.Environment); err != nil {
			return fmt.Errorf("write health: %w", err)
		}
	}

	return nil
}
