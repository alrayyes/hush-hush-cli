package cmd

import (
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newDeleteCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Permanently remove an object",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			if err := cli.Delete(cmd.Context(), cfg, args[0]); err != nil {
				return fmt.Errorf("delete: %w", err)
			}

			return nil
		},
	}
}
