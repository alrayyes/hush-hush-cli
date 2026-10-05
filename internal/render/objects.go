package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// ObjectsJSON prints objects as one JSON array, `[]` for none.
func ObjectsJSON(out io.Writer, objects []client.ObjectMetadata) error {
	if objects == nil {
		objects = []client.ObjectMetadata{}
	}

	if err := json.NewEncoder(out).Encode(objects); err != nil {
		return fmt.Errorf("write list as json: %w", err)
	}

	return nil
}

// ObjectsTable prints objects as an aligned table, "-" for a time the server left out.
func ObjectsTable(out io.Writer, objects []client.ObjectMetadata) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintln(w, "ID\tUSED BY\tTAGS\tCREATED\tUPDATED\tDESCRIPTION"); err != nil {
		return fmt.Errorf("write list header: %w", err)
	}

	for _, obj := range objects {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", obj.Slug, strings.Join(obj.UsedBy, ","), strings.Join(obj.Tags, ","), formatTime(obj.CreatedAt), formatTime(obj.UpdatedAt), obj.Description); err != nil {
			return fmt.Errorf("write list row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush list table: %w", err)
	}

	return nil
}

// formatTime renders t in local time, or "-" when the server sent none.
func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}

	return t.Local().Format(time.DateTime)
}
