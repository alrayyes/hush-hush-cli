package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// StatusJSON prints the auth status as JSON.
func StatusJSON(out io.Writer, status client.AuthStatus) error {
	if err := json.NewEncoder(out).Encode(status); err != nil {
		return fmt.Errorf("write status as json: %w", err)
	}

	return nil
}
