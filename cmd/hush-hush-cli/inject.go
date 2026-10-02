package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// errNoRecipients is a sentinel rather than a plain fmt.Errorf: it's a
// fixed condition (recipients weren't configured at all), not a message
// built from per-call detail.
var errNoRecipients = errors.New("no recipients configured (--recipients, HUSH_HUSH_RECIPIENTS, or run `hush-hush-cli init`)")

// errNoRecipientsOrUsedBy is inject's own version of errNoRecipients:
// unlike update, inject can still resolve recipients from --used-by's
// registered consumer keys, so the message names that path too.
var errNoRecipientsOrUsedBy = errors.New("no recipients configured (--recipients, HUSH_HUSH_RECIPIENTS, --used-by naming a consumer with a registered public key, or run `hush-hush-cli init`)")

// errTagAndClearTags is a sentinel: a fixed condition (both flags given).
var errTagAndClearTags = errors.New("--tag and --clear-tags can't be combined")

// tagFlagUsage is shared by inject and update's --tag.
const tagFlagUsage = "label for grouping secrets (repeatable, or comma-separated; 1-32 chars of a-z 0-9 . _ / -, max 10)"

// newInjectCmd reads the plaintext value from stdin rather than a flag -
// a flag value ends up in shell history and process listings, exactly
// what injecting a secret should avoid.
func newInjectCmd() *cobra.Command {
	var (
		usedBy      []string
		tags        []string
		description string
	)

	cmd := &cobra.Command{
		Use:   "inject <id>",
		Short: "Seal a value (read from stdin) and create a new object",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Rebound here rather than at construction: inject and update
			// share the --recipients/HUSH_HUSH_RECIPIENTS name, and binding
			// both flag objects to the one viper key at construction time
			// leaves viper pointing at whichever command was registered
			// last on root, silently ignoring the other's flag.
			_ = viper.BindPFlag("recipients", cmd.Flags().Lookup("recipients"))

			recipients := viper.GetString("recipients")
			if recipients == "" && len(usedBy) == 0 {
				return errNoRecipientsOrUsedBy
			}

			var recipientList []string
			if recipients != "" {
				recipientList = strings.Split(recipients, ",")
			}

			value, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read value from stdin: %w", err)
			}

			cfg, err := config(true)
			if err != nil {
				return err
			}

			return cli.Inject(cmd.Context(), cfg, args[0], value, recipientList, usedBy, description, cli.WithTags(tags))
		},
	}

	cmd.Flags().StringSliceVar(&usedBy, "used-by", nil, "consumers of this secret (repeatable, or comma-separated)")
	cmd.Flags().String("recipients", "", "comma-separated age recipient public keys")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, tagFlagUsage)
	cmd.Flags().StringVar(&description, "description", "", "free-text label for this object, fixed at creation")

	return cmd
}
