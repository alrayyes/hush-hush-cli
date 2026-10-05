package cmd

import (
	"errors"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/render"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// errUnknownAuditLogFormat is a sentinel: a fixed condition (an
// unsupported --format value), not a message built from per-call detail -
// the value itself is per-call detail, wrapped in at the call site.
var errUnknownAuditLogFormat = errors.New("must be table or json")

// newAuditLogCmd makes exactly one request per invocation and exits -
// spec.md's "No live/follow mode" requirement, matching gh's own
// audit-log command shape.
func newAuditLogCmd(v *viper.Viper) *cobra.Command {
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

			filter, err := cli.ParseAuditLogFilter(objectID, actor, caller, since, until, limit)
			if err != nil {
				return err //nolint:wrapcheck // the error already names the flag
			}

			cfg, err := config(v, false)
			if err != nil {
				return err
			}

			entries, err := cli.AuditLog(cmd.Context(), cfg, filter)
			if err != nil {
				return fmt.Errorf("audit-log: %w", err)
			}

			if format == "json" {
				return render.AuditLogJSON(cmd.OutOrStdout(), entries)
			}

			return render.AuditLogTable(cmd.OutOrStdout(), entries)
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
