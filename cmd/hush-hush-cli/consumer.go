package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/spf13/cobra"
)

// errConsumerUpdateNothing is a sentinel: a fixed condition (neither flag
// given), not a message built from per-call detail.
var errConsumerUpdateNothing = errors.New("nothing to update: pass --name and/or --public-key")

// newConsumerCmd groups the consumer directory commands - all of them
// require a write token, the same as list: the directory enumerates names
// no caller necessarily holds the id of.
func newConsumerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consumer",
		Short: "Manage the consumer directory",
	}

	cmd.AddCommand(newConsumerListCmd())
	cmd.AddCommand(newConsumerAddCmd())
	cmd.AddCommand(newConsumerUpdateCmd())
	cmd.AddCommand(newConsumerDeleteCmd())

	return cmd
}

func newConsumerListCmd() *cobra.Command {
	var (
		query  string
		asJSON bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List consumers with their secret count and public key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config(true)
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

				return writeConsumerJSON(cmd, consumers)
			}

			return writeConsumerTable(cmd, consumers)
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "only consumers whose name contains this text (case-insensitive)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON array instead of a table")

	return cmd
}

func newConsumerAddCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a consumer no secret references yet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config(true)
			if err != nil {
				return err
			}

			consumer, err := cli.AddConsumer(cmd.Context(), cfg, args[0])
			if err != nil {
				return fmt.Errorf("consumer add: %w", err)
			}

			return writeOneConsumer(cmd, consumer, asJSON)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON object instead of a table")

	return cmd
}

func newConsumerUpdateCmd() *cobra.Command {
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

			cfg, err := config(true)
			if err != nil {
				return err
			}

			consumer, err := cli.UpdateConsumer(cmd.Context(), cfg, args[0], namePtr, keyPtr)
			if err != nil {
				return fmt.Errorf("consumer update: %w", err)
			}

			return writeOneConsumer(cmd, consumer, asJSON)
		},
	}

	cmd.Flags().StringVar(&newName, "name", "", "the consumer's new name (merges into an existing one of that name)")
	cmd.Flags().StringVar(&publicKey, "public-key", "", "the consumer's age public key (age1...)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON object instead of a table")

	return cmd
}

func newConsumerDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Remove a consumer from every secret that references it",
		Long: "Strips the consumer from every secret's used_by list and the directory. " +
			"No secret is deleted, even one left with an empty used_by list.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config(true)
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

func writeOneConsumer(cmd *cobra.Command, consumer client.Consumer, asJSON bool) error {
	if asJSON {
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(consumer); err != nil {
			return fmt.Errorf("write consumer as json: %w", err)
		}

		return nil
	}

	return writeConsumerTable(cmd, []client.Consumer{consumer})
}

func writeConsumerJSON(cmd *cobra.Command, consumers []client.Consumer) error {
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(consumers); err != nil {
		return fmt.Errorf("write consumers as json: %w", err)
	}

	return nil
}

func writeConsumerTable(cmd *cobra.Command, consumers []client.Consumer) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintln(w, "NAME\tSECRETS\tPUBLIC KEY"); err != nil {
		return fmt.Errorf("write consumer header: %w", err)
	}

	for _, c := range consumers {
		if _, err := fmt.Fprintf(w, "%s\t%d\t%s\n", c.Name, c.SecretCount, c.PublicKey); err != nil {
			return fmt.Errorf("write consumer row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush consumer table: %w", err)
	}

	return nil
}
