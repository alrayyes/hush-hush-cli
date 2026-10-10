package cmd

import (
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/render"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newHealthCmd is unauthenticated like status: a probe or deploy script
// with no token still gets an answer. A server that is down is an error
// (non-zero exit), unlike status's unbootstrapped case, so a script can
// wait on it.
func newHealthCmd(v *viper.Viper) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Report whether the target server is up",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config(v, false)
			if err != nil {
				return err
			}

			health, err := cli.Health(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("health: %w", err)
			}

			if asJSON {
				return render.HealthJSON(cmd.OutOrStdout(), health)
			}

			return render.HealthText(cmd.OutOrStdout(), health)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON object instead of plain lines")

	return cmd
}
