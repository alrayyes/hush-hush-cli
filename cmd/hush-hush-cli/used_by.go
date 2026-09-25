package main

import (
	"encoding/json"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
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
				return writeUsedByJSON(cmd, usedBy)
			}

			return writeUsedByLines(cmd, usedBy)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON array instead of one consumer per line")

	return cmd
}

func writeUsedByJSON(cmd *cobra.Command, usedBy []string) error {
	if usedBy == nil {
		usedBy = []string{}
	}

	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(usedBy); err != nil {
		return fmt.Errorf("write used-by as json: %w", err)
	}

	return nil
}

func writeUsedByLines(cmd *cobra.Command, usedBy []string) error {
	for _, consumer := range usedBy {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), consumer); err != nil {
			return fmt.Errorf("write used-by line: %w", err)
		}
	}

	return nil
}
