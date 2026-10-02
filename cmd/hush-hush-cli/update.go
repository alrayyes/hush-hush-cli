package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newUpdateCmd reads the new plaintext value from stdin, same reasoning as
// newInjectCmd: a flag value ends up in shell history and process listings.
func newUpdateCmd() *cobra.Command {
	var (
		tags      []string
		clearTags bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Seal a new value (read from stdin) and replace an object's stored value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// See newInjectCmd: rebound here, not at construction, since
			// inject and update share the --recipients/HUSH_HUSH_RECIPIENTS
			// name and only one command's flag object can hold the viper
			// key at a time.
			_ = viper.BindPFlag("recipients", cmd.Flags().Lookup("recipients"))

			if clearTags && cmd.Flags().Changed("tag") {
				return errTagAndClearTags
			}

			recipients := viper.GetString("recipients")
			if recipients == "" {
				return errNoRecipients
			}

			value, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read value from stdin: %w", err)
			}

			cfg, err := config(true)
			if err != nil {
				return err
			}

			var opts []cli.WriteOption

			switch {
			case clearTags:
				opts = append(opts, cli.WithTags([]string{}))
			case cmd.Flags().Changed("tag"):
				opts = append(opts, cli.WithTags(tags))
			}

			return cli.Update(cmd.Context(), cfg, args[0], value, strings.Split(recipients, ","), opts...)
		},
	}

	cmd.Flags().String("recipients", "", "comma-separated age recipient public keys")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, tagFlagUsage+"; replaces the object's tags")
	cmd.Flags().BoolVar(&clearTags, "clear-tags", false, "remove every tag from the object")

	return cmd
}
