package cmd

import (
	"errors"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var errNothingToEdit = errors.New("nothing to edit: give at least one of --description, --tag, --clear-tags or --slug")

// newEditCmd changes an object's metadata only. Unlike update it reads no
// value from stdin and seals nothing.
func newEditCmd(v *viper.Viper) *cobra.Command {
	var (
		f           variantFlags
		description string
		slug        string
		tags        []string
		clearTags   bool
	)

	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Change an object's description, tags or name without touching its value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.validate(); err != nil {
				return err
			}

			changed := cmd.Flags().Changed

			switch {
			case clearTags && changed("tag"):
				return errTagAndClearTags
			case !changed("description") && !changed("slug") && !changed("tag") && !clearTags:
				return errNothingToEdit
			}

			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			return cli.Edit(cmd.Context(), cfg, args[0], editOptions(cmd, f, description, slug, tags, clearTags)...)
		},
	}

	f.register(cmd)
	cmd.Flags().StringVar(&description, "description", "", "replace the free-text label (empty clears it)")
	cmd.Flags().StringVar(&slug, "slug", "", "rename the object; every variant of the name moves")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, tagFlagUsage+"; replaces the object's tags")
	cmd.Flags().BoolVar(&clearTags, "clear-tags", false, "remove every tag from the object")

	return cmd
}

// editOptions turns the flags the user actually gave into Edit's options; a
// flag left out is not sent, so the server leaves that field alone.
func editOptions(cmd *cobra.Command, f variantFlags, description, slug string, tags []string, clearTags bool) []cli.WriteOption {
	changed := cmd.Flags().Changed

	var opts []cli.WriteOption

	if changed("description") {
		opts = append(opts, cli.WithDescription(description))
	}

	if changed("slug") {
		opts = append(opts, cli.WithSlug(slug))
	}

	switch {
	case clearTags:
		opts = append(opts, cli.WithTags([]string{}))
	case changed("tag"):
		opts = append(opts, cli.WithTags(tags))
	}

	if f.id != "" {
		opts = append(opts, cli.WithVariant(f.id))
	}

	if f.consumer != "" {
		opts = append(opts, cli.WithConsumer(f.consumer))
	}

	return opts
}
