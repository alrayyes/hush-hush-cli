package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

// keyringService names hush-hush-cli's own OS keyring entries - one
// "account" per credential field ("token", "identity").
const keyringService = "hush-hush-cli"

// newConfigCmd groups init's own generated _command subcommands under one
// hidden parent, rather than a bare top-level "keyring-get": nothing here
// is meant to be typed by hand, only shelled out to by a token_command/
// identity_command the interactive init wrote.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "config",
		Short:  "Config-file internals (not for direct use)",
		Hidden: true,
	}

	cmd.AddCommand(newKeyringGetCmd())

	return cmd
}

// newKeyringGetCmd is what "hush-hush-cli config keyring-get <field>"
// runs - the command an interactive init generates for the OS-keyring
// persistence choice (rules/cli.md's "secrets get a command option, not
// just a value"). Reading it back through runSecretCommand rather than
// calling keyring.Get directly from config() keeps every credential field
// resolving through the exact same literal-or-command path regardless of
// where the command form points.
func newKeyringGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "keyring-get <field>",
		Short:  "Print a value previously stored in the OS keyring (not for direct use)",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := keyring.Get(keyringService, args[0])
			if err != nil {
				return fmt.Errorf("keyring get %s: %w", args[0], err)
			}

			if _, err := fmt.Fprintln(cmd.OutOrStdout(), value); err != nil {
				return fmt.Errorf("write keyring value: %w", err)
			}

			return nil
		},
	}
}
