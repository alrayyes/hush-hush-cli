package cmd

import (
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newDeleteCmd(v *viper.Viper) *cobra.Command {
	var variantID string

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Permanently remove an object",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			if err := cli.Delete(cmd.Context(), cfg, args[0], client.WithVariant(variantID)); err != nil {
				return fmt.Errorf("delete: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&variantID, "id", "", variantIDUsage)

	return cmd
}
