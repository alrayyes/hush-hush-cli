package cmd

import (
	"errors"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/render"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// errConsumerUpdateNothing is a sentinel: a fixed condition (neither flag
// given), not a message built from per-call detail.
var errConsumerUpdateNothing = errors.New("nothing to update: pass --name and/or --public-key")

// newConsumerCmd groups the consumer directory commands - all of them
// require a write token, the same as list: the directory enumerates names
// no caller necessarily holds the id of.
func newConsumerCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consumer",
		Short: "Manage the consumer directory",
	}

	cmd.AddCommand(newConsumerListCmd(v))
	cmd.AddCommand(newConsumerAddCmd(v))
	cmd.AddCommand(newConsumerUpdateCmd(v))
	cmd.AddCommand(newConsumerDeleteCmd(v))

	return cmd
}

func newConsumerListCmd(v *viper.Viper) *cobra.Command {
	var (
		query  string
		asJSON bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List consumers with their secret count and public key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			consumers, err := cli.ListConsumers(cmd.Context(), cfg, query)
			if err != nil {
				return fmt.Errorf("consumer list: %w", err)
			}

			if asJSON {
				if consumers == nil {
					consumers = []client.Consumer{}
				}

				return render.ConsumersJSON(cmd.OutOrStdout(), consumers)
			}

			return render.ConsumersTable(cmd.OutOrStdout(), consumers)
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "only consumers whose name contains this text (case-insensitive)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON array instead of a table")

	return cmd
}

func newConsumerAddCmd(v *viper.Viper) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a consumer no secret references yet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			consumer, err := cli.AddConsumer(cmd.Context(), cfg, args[0])
			if err != nil {
				return fmt.Errorf("consumer add: %w", err)
			}

			return render.OneConsumer(cmd.OutOrStdout(), consumer, asJSON)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON object instead of a table")

	return cmd
}

func newConsumerUpdateCmd(v *viper.Viper) *cobra.Command {
	var (
		newName   string
		publicKey string
		asJSON    bool
	)

	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Rename a consumer and/or register its age public key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var namePtr, keyPtr *string

			if cmd.Flags().Changed("name") {
				namePtr = &newName
			}

			if cmd.Flags().Changed("public-key") {
				keyPtr = &publicKey
			}

			if namePtr == nil && keyPtr == nil {
				return errConsumerUpdateNothing
			}

			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			consumer, err := cli.UpdateConsumer(cmd.Context(), cfg, args[0], namePtr, keyPtr)
			if err != nil {
				return fmt.Errorf("consumer update: %w", err)
			}

			return render.OneConsumer(cmd.OutOrStdout(), consumer, asJSON)
		},
	}

	cmd.Flags().StringVar(&newName, "name", "", "the consumer's new name (merges into an existing one of that name)")
	cmd.Flags().StringVar(&publicKey, "public-key", "", "the consumer's age public key (age1...)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON object instead of a table")

	return cmd
}

func newConsumerDeleteCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Remove a consumer from every secret that references it",
		Long: "Strips the consumer from every secret's used_by list and the directory. " +
			"No secret is deleted, even one left with an empty used_by list.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			if err := cli.DeleteConsumer(cmd.Context(), cfg, args[0]); err != nil {
				return fmt.Errorf("consumer delete: %w", err)
			}

			return nil
		},
	}
}
