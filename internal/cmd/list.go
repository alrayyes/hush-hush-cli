package cmd

import (
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/render"
	"github.com/spf13/cobra"
)

// newListCmd requires a token, unlike get: enumerating every stored object
// is a capability none of the other, id-scoped reads grant on their own,
// matching hush-hush's own GET /objects.
func newListCmd() *cobra.Command {
	var (
		asJSON bool
		usedBy string
		tags   []string
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

			objects, err := cli.List(cmd.Context(), cfg, usedBy, tags)
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}

			if asJSON {
				return render.ObjectsJSON(cmd.OutOrStdout(), objects)
			}

			return render.ObjectsTable(cmd.OutOrStdout(), objects)
		},
	}

	cmd.Flags().StringVar(&usedBy, "used-by", "", "only objects whose used_by includes this consumer")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "only objects carrying this tag (repeatable, or comma-separated; all must match)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON array instead of a table")

	return cmd
}
