package render

import (
	"encoding/json"
	"fmt"
	"io"
)

// UsedByJSON prints consumer names as one JSON array, `[]` for none.
func UsedByJSON(out io.Writer, usedBy []string) error {
	if usedBy == nil {
		usedBy = []string{}
	}

	if err := json.NewEncoder(out).Encode(usedBy); err != nil {
		return fmt.Errorf("write used-by as json: %w", err)
	}

	return nil
}

// UsedByLines prints one consumer name per line.
func UsedByLines(out io.Writer, usedBy []string) error {
	for _, consumer := range usedBy {
		if _, err := fmt.Fprintln(out, consumer); err != nil {
			return fmt.Errorf("write used-by line: %w", err)
		}
	}

	return nil
}
