package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/spf13/cobra"
)

// newListCmd requires a token, unlike get: enumerating every stored object
// is a capability none of the other, id-scoped reads grant on their own,
// matching hush-hush's own GET /objects.
func newListCmd() *cobra.Command {
	var (
		asJSON bool
		usedBy string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored objects' metadata",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config(true)
			if err != nil {
				return err
			}

			objects, err := cli.List(cmd.Context(), cfg, usedBy)
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			if asJSON {
				return writeListJSON(cmd, objects)
			}

			return writeListTable(cmd, objects)
		},
	}

	cmd.Flags().StringVar(&usedBy, "used-by", "", "only objects whose used_by includes this consumer")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON array instead of a table")

	return cmd
}

func writeListJSON(cmd *cobra.Command, objects []client.ObjectMetadata) error {
	if objects == nil {
		objects = []client.ObjectMetadata{}
	}

	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(objects); err != nil {
		return fmt.Errorf("write list as json: %w", err)
	}

	return nil
}

func writeListTable(cmd *cobra.Command, objects []client.ObjectMetadata) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)

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
