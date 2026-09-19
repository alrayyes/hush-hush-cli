package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/spf13/cobra"
)

// errUnknownAuditLogFormat is a sentinel: a fixed condition (an
// unsupported --format value), not a message built from per-call detail -
// the value itself is per-call detail, wrapped in at the call site.
var errUnknownAuditLogFormat = errors.New("must be table or json")

// newAuditLogCmd makes exactly one request per invocation and exits -
// spec.md's "No live/follow mode" requirement, matching gh's own
// audit-log command shape.
func newAuditLogCmd() *cobra.Command {
	var (
		objectID, actor, caller, since, until, format string
		limit                                         int
	)

	cmd := &cobra.Command{
		Use:   "audit-log",
		Short: "Query the audit trail of create/read/update/delete calls",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != "table" && format != "json" {
				return fmt.Errorf("--format %q: %w", format, errUnknownAuditLogFormat)
			}

			filter, err := buildAuditLogFilter(objectID, actor, caller, since, until, limit)
			if err != nil {
				return err
			}

			cfg, err := config(false)
			if err != nil {
				return err
			}

			entries, err := cli.AuditLog(cmd.Context(), cfg, filter)
			if err != nil {
				return fmt.Errorf("audit-log: %w", err)
			}

			if format == "json" {
				return writeAuditLogJSON(cmd, entries)
			}

			return writeAuditLogTable(cmd, entries)
		},
	}

	cmd.Flags().StringVar(&objectID, "object", "", "restrict to entries for this object id")
	cmd.Flags().StringVar(&actor, "actor", "",
		"restrict to entries authenticated by this verified actor (a token id, or the admin account's own actor id)")
	cmd.Flags().StringVar(&caller, "caller", "", "restrict to entries recorded with this caller identity")
	cmd.Flags().StringVar(&since, "since", "", "restrict to entries at or after this RFC3339 time")
	cmd.Flags().StringVar(&until, "until", "", "restrict to entries at or before this RFC3339 time")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table or json")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of entries to print (0 means every matching entry)")

	return cmd
}

// buildAuditLogFilter parses --since/--until as RFC3339 (design.md) before
// any request is made, so a malformed value fails locally rather than as
// a 400 from the server.
func buildAuditLogFilter(objectID, actor, caller, since, until string, limit int) (client.AuditLogFilter, error) {
	var filter client.AuditLogFilter

	if objectID != "" {
		filter.ObjectID = &objectID
	}

	if actor != "" {
		filter.Token = &actor
	}

	if caller != "" {
		filter.Caller = &caller
	}

	if since != "" {
		t, err := time.Parse(time.RFC3339, since)
		if err != nil {
			return filter, fmt.Errorf("--since: %w", err)
		}

		filter.Since = &t
	}

	if until != "" {
		t, err := time.Parse(time.RFC3339, until)
		if err != nil {
			return filter, fmt.Errorf("--until: %w", err)
		}

		filter.Until = &t
	}

	if limit > 0 {
		filter.Limit = &limit
	}

	return filter, nil
}

func writeAuditLogJSON(cmd *cobra.Command, entries []client.AuditLogEntry) error {
	if entries == nil {
		entries = []client.AuditLogEntry{}
	}

	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(entries); err != nil {
		return fmt.Errorf("write audit-log as json: %w", err)
	}

	return nil
}

func writeAuditLogTable(cmd *cobra.Command, entries []client.AuditLogEntry) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)

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
