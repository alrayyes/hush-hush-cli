package cmd

import (
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/render"
	"github.com/spf13/cobra"
)

// newUsedByCmd is unauthenticated, matching newGetCmd: fetching one
// already-known object's recorded consumers needs no token, unlike list's
// full enumeration.
func newUsedByCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "used-by <id>",
		Short: "Print an object's recorded consumers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config(false)
			if err != nil {
				return err
			}

			usedBy, err := cli.UsedBy(cmd.Context(), cfg, args[0])
			if err != nil {
				return fmt.Errorf("used-by: %w", err)
			}

			if asJSON {
				return render.UsedByJSON(cmd.OutOrStdout(), usedBy)
			}

			return render.UsedByLines(cmd.OutOrStdout(), usedBy)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON array instead of one consumer per line")

	return cmd
}
