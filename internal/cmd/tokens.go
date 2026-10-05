package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/render"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// errTTLRequired is a sentinel: a fixed condition (no --ttl given), not a
// message built from per-call detail.
var errTTLRequired = errors.New("--ttl is required (a Go duration, e.g. 720h)")

// newTokenCmd groups every consumer read token lifecycle command - all of
// them require a write token, the same as inject/update/delete: minting
// or managing a credential that grants read access is itself a
// write-path operation (alrayyes/hush-hush#467).
func newTokenCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Mint and manage consumer read tokens",
	}

	cmd.AddCommand(newTokenCreateCmd(v))
	cmd.AddCommand(newTokenListCmd(v))
	cmd.AddCommand(newTokenRotateCmd(v))
	cmd.AddCommand(newTokenRevokeCmd(v))
	cmd.AddCommand(newTokenPurgeCmd(v))

	return cmd
}

func newTokenCreateCmd(v *viper.Viper) *cobra.Command {
	var (
		description string
		ttl         string
		asJSON      bool
	)

	cmd := &cobra.Command{
		Use:   "create <consumer>",
		Short: "Mint a new consumer read token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := parseTTL(ttl)
			if err != nil {
				return err
			}

			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			token, err := cli.CreateConsumerToken(cmd.Context(), cfg, args[0], description, d)
			if err != nil {
				return fmt.Errorf("token create: %w", err)
			}

			return render.ConsumerTokenWithValue(cmd.OutOrStdout(), token, asJSON)
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "free-text label for the token")
	cmd.Flags().StringVar(&ttl, "ttl", "", "how long the token stays valid, starting now (a Go duration, e.g. 720h)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON object instead of a table")

	return cmd
}

func newTokenListCmd(v *viper.Viper) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List every issued consumer token's metadata",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			tokens, err := cli.ListConsumerTokens(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("token list: %w", err)
			}

			if asJSON {
				return render.ConsumerTokensJSON(cmd.OutOrStdout(), tokens)
			}

			return render.ConsumerTokensTable(cmd.OutOrStdout(), tokens)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON array instead of a table")

	return cmd
}

func newTokenRotateCmd(v *viper.Viper) *cobra.Command {
	var (
		ttl    string
		asJSON bool
	)

	cmd := &cobra.Command{
		Use:   "rotate <id>",
		Short: "Replace a consumer token's secret and expiry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := parseTTL(ttl)
			if err != nil {
				return err
			}

			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			token, err := cli.RotateConsumerToken(cmd.Context(), cfg, args[0], d)
			if err != nil {
				return fmt.Errorf("token rotate: %w", err)
			}

			return render.ConsumerTokenWithValue(cmd.OutOrStdout(), token, asJSON)
		},
	}

	cmd.Flags().StringVar(&ttl, "ttl", "", "how long the rotated token stays valid, starting now (a Go duration, e.g. 720h)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the raw JSON object instead of a table")

	return cmd
}

func newTokenRevokeCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <id>",
		Short: "Invalidate a consumer token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			if err := cli.RevokeConsumerToken(cmd.Context(), cfg, args[0]); err != nil {
				return fmt.Errorf("token revoke: %w", err)
			}

			return nil
		},
	}
}

func newTokenPurgeCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "purge <id>",
		Short: "Permanently remove an already-revoked or expired consumer token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config(v, true)
			if err != nil {
				return err
			}

			if err := cli.PurgeConsumerToken(cmd.Context(), cfg, args[0]); err != nil {
				return fmt.Errorf("token purge: %w", err)
			}

			return nil
		},
	}
}

func parseTTL(ttl string) (time.Duration, error) {
	if ttl == "" {
		return 0, errTTLRequired
	}

	d, err := time.ParseDuration(ttl)
	if err != nil {
		return 0, fmt.Errorf("--ttl: %w", err)
	}

	return d, nil
}
