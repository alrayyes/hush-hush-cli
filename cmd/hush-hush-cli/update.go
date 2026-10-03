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
	var f updateFlags

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

			recipients, err := f.recipients(cmd)
			if err != nil {
				return err
			}

			value, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read value from stdin: %w", err)
			}

			cfg, err := config(true)
			if err != nil {
				return err
			}

			return cli.Update(cmd.Context(), cfg, args[0], value, recipients, f.options(cmd)...)
		},
	}

	cmd.Flags().String("recipients", "", "comma-separated age recipient public keys")
	cmd.Flags().StringSliceVar(&f.tags, "tag", nil, tagFlagUsage+"; replaces the object's tags")
	cmd.Flags().BoolVar(&f.clearTags, "clear-tags", false, "remove every tag from the object")
	cmd.Flags().StringSliceVar(&f.usedBy, "used-by", nil, "replace the object's consumers (repeatable, or comma-separated); seals to their registered keys unless --recipients is given")
	cmd.Flags().BoolVar(&f.clearUsedBy, "clear-used-by", false, "remove every consumer from the object")

	return cmd
}

// updateFlags holds update's tag and consumer flags.
type updateFlags struct {
	tags        []string
	clearTags   bool
	usedBy      []string
	clearUsedBy bool
}

// recipients validates the flag combination and returns the explicit
// recipients, nil when they should be resolved from --used-by instead.
func (f updateFlags) recipients(cmd *cobra.Command) ([]string, error) {
	usedByGiven := cmd.Flags().Changed("used-by")

	switch {
	case f.clearTags && cmd.Flags().Changed("tag"):
		return nil, errTagAndClearTags
	case f.clearUsedBy && usedByGiven:
		return nil, errUsedByAndClearUsedBy
	}

	recipients := viper.GetString("recipients")

	switch {
	case recipients != "":
		return strings.Split(recipients, ","), nil
	case len(f.usedBy) > 0:
		return nil, nil
	case usedByGiven:
		return nil, errNoRecipientsOrUsedBy
	default:
		return nil, errNoRecipients
	}
}

func (f updateFlags) options(cmd *cobra.Command) []cli.WriteOption {
	var opts []cli.WriteOption

	switch {
	case f.clearTags:
		opts = append(opts, cli.WithTags([]string{}))
	case cmd.Flags().Changed("tag"):
		opts = append(opts, cli.WithTags(f.tags))
	}

	switch {
	case f.clearUsedBy:
		opts = append(opts, cli.WithUsedBy([]string{}))
	case cmd.Flags().Changed("used-by"):
		opts = append(opts, cli.WithUsedBy(f.usedBy))
	}

	return opts
}
