package render

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// AuditLogJSON prints audit entries as one JSON array, `[]` for none.
func AuditLogJSON(out io.Writer, entries []client.AuditLogEntry) error {
	if entries == nil {
		entries = []client.AuditLogEntry{}
	}

	if err := json.NewEncoder(out).Encode(entries); err != nil {
		return fmt.Errorf("write audit-log as json: %w", err)
	}

	return nil
}

// AuditLogTable prints audit entries as an aligned table, "-" for a missing caller.
func AuditLogTable(out io.Writer, entries []client.AuditLogEntry) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintln(w, "TIMESTAMP\tACTION\tOBJECT ID\tCALLER\tIP"); err != nil {
		return fmt.Errorf("write audit-log header: %w", err)
	}

	for _, e := range entries {
		caller := "-"
		if e.Caller != nil {
			caller = *e.Caller
		}

		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.Timestamp.Local().Format(time.DateTime), e.Action, e.ObjectID, caller, e.IP); err != nil {
			return fmt.Errorf("write audit-log row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush audit-log table: %w", err)
	}

	return nil
}
