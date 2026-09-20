package main

import (
	"encoding/json"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/spf13/cobra"
)

// newStatusCmd is unauthenticated, matching internal/cli.AuthStatus - it
// works with no token configured at all, unlike list/inject/update/delete.
// An unbootstrapped server is a normal result to print, not an error: exit
// 0 either way (#107's own acceptance criteria).
func newStatusCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report whether the target server has an admin account bootstrapped yet",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config(false)
			if err != nil {
				return err
			}

			status, err := cli.AuthStatus(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("status: %w", err)
			}

			if asJSON {
				return writeStatusJSON(cmd, status)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "bootstrapped: %t\n", status.Bootstrapped)
			if err != nil {
				return fmt.Errorf("write status: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON object instead of a plain line")

	return cmd
}

func writeStatusJSON(cmd *cobra.Command, status client.AuthStatus) error {
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(status); err != nil {
		return fmt.Errorf("write status as json: %w", err)
	}

	return nil
}
