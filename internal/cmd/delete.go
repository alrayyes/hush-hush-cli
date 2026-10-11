package cmd

import (
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newDeleteCmd(v *viper.Viper) *cobra.Command {
	var variant variantFlags

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Permanently remove an object",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := variant.validate(); err != nil {
				return err
			}

			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			if err := cli.Delete(cmd.Context(), cfg, args[0], variant.callOptions()...); err != nil {
				return fmt.Errorf("delete: %w", err)
			}

			return nil
		},
	}

	variant.register(cmd)

	return cmd
}
