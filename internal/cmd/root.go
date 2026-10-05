package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/cliconfig"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// errConfigAlreadyExists is a sentinel rather than a plain fmt.Errorf: a
// fixed condition (a file is already there), not a message built from
// per-call detail - the path itself is per-call detail, so it's wrapped
// in rather than folded into the message.
var errConfigAlreadyExists = errors.New("config file already exists (use --force to overwrite)")

// Option configures NewRootCmd.
type Option func(*options)

type options struct {
	configPath string
}

// pathResolver is the config-file path lookup: the fixed path WithConfigPath
// gave, else the one under XDG_CONFIG_HOME.
func (o options) pathResolver() func() (string, error) {
	if o.configPath == "" {
		return configFilePath
	}

	return func() (string, error) { return o.configPath, nil }
}

// WithConfigPath reads and writes the config file at path instead of the one
// under XDG_CONFIG_HOME. A test that uses it needs no t.Setenv, so it can
// run in parallel.
func WithConfigPath(path string) Option {
	return func(o *options) { o.configPath = path }
}

// NewRootCmd wires the persistent, env-overridable connection config
// (server URL, bearer token, caller identity) shared by every subcommand.
// HUSH_HUSH_SERVER, HUSH_HUSH_TOKEN, and HUSH_HUSH_CALLER override their
// matching flags - a CI job supplies these through its own secret storage,
// with no bespoke wrapper or Action (the cli spec's "runs unmodified
// inside CI" requirement). A config file at configPath() sits below both:
// rules/cli.md's flags > environment > config file > defaults.
func NewRootCmd(version string, opts ...Option) *cobra.Command {
	root, _ := newRootCmd(version, opts...)

	return root
}

// newRootCmd is NewRootCmd plus the viper instance every subcommand reads,
// for the internal tests that call config() directly.
func newRootCmd(version string, opts ...Option) (*cobra.Command, *viper.Viper) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	resolvePath := o.pathResolver()

	// One viper per command tree, not the package-level one: two roots (two
	// tests, say) must never see each other's flags, env or config file.
	v := viper.New()

	root := &cobra.Command{
		Use:           "hush-hush-cli",
		Short:         "Client for the hush-hush secrets object store",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// man generates pages by walking the command tree, not by
			// connecting to a server - the same reason init is exempt.
			// config/keyring-get is what a keyring-persisted _command
			// shells out to - it must never itself trigger the nudge, or
			// a token_command pointing at it would recurse into prompting
			// for setup every time something resolves the token.
			switch cmd.Name() {
			case "init", "man", "config", "keyring-get":
				return nil
			default:
				return maybeOfferInit(v, resolvePath, cmd)
			}
		},
	}

	root.PersistentFlags().String("server", "http://localhost:8080", "hush-hush server URL")
	root.PersistentFlags().String("token", "", "write-path bearer token")
	root.PersistentFlags().String("token-command", "", "command whose trimmed stdout is the write-path bearer token (wins over --token if both are set)")
	root.PersistentFlags().String("consumer-token", "", "read-only, consumer-scoped bearer token - used by get only, as a fallback when --token isn't set")
	root.PersistentFlags().String("consumer-token-command", "", "command whose trimmed stdout is the consumer token instead (wins over --consumer-token if both are set)")
	root.PersistentFlags().String("caller", "", "self-presented identity recorded in the audit log")
	root.PersistentFlags().BoolP("yes", "y", false, "write a starter config with no prompt, if none exists")

	bindConnectionFlags(root, v)

	v.SetEnvPrefix("hush_hush")
	v.AutomaticEnv()

	if path, err := resolvePath(); err == nil {
		v.SetConfigFile(path)
		v.SetConfigType("yaml")
		_ = v.ReadInConfig() // no config file yet is not an error
	}

	root.AddCommand(newInitCmd(v, resolvePath))
	root.AddCommand(newInjectCmd(v))
	root.AddCommand(newGetCmd(v))
	root.AddCommand(newUpdateCmd(v))
	root.AddCommand(newListCmd(v))
	root.AddCommand(newDeleteCmd(v))
	root.AddCommand(newUsedByCmd(v))
	root.AddCommand(newAuditLogCmd(v))
	root.AddCommand(newTokenCmd(v))
	root.AddCommand(newConsumerCmd(v))
	root.AddCommand(newStatusCmd(v))
	root.AddCommand(newConfigCmd())
	root.AddCommand(newManCmd(root))

	return root, v
}

// bindConnectionFlags binds the persistent connection flags to their config
// keys, so a flag, an environment variable and a config-file key all name
// the same setting.
func bindConnectionFlags(root *cobra.Command, v *viper.Viper) {
	for _, name := range []string{"server", "token", "caller"} {
		_ = v.BindPFlag(name, root.PersistentFlags().Lookup(name))
	}

	_ = v.BindPFlag("token_command", root.PersistentFlags().Lookup("token-command"))
	_ = v.BindPFlag("consumer_token", root.PersistentFlags().Lookup("consumer-token"))
	_ = v.BindPFlag("consumer_token_command", root.PersistentFlags().Lookup("consumer-token-command"))
}

// config resolves the CLI's connection settings, running --token-command/
// HUSH_HUSH_TOKEN_COMMAND if set - rules/cli.md's "secrets get a command
// option, not just a value", so a token can come from `pass`, an
// age-encrypted file, or a keyring CLI instead of sitting in the config
// file as plaintext. The command wins over a literal --token/token if both
// are set: whoever configured the command form did it on purpose.
//
// consumer_token/consumer_token_command resolve the same way, for get's
// fallback read-only credential (internal/cli.Get, design.md).
//
// requireToken is set by inject/update/delete and left false by get: a
// missing token then fails Validate() here, before any request reaches
// the server, rather than surfacing as a bare 401 from deep inside the
// SDK. get accepts either token being empty - it falls back to whichever
// of Token/ConsumerToken is set, or neither, itself.
func config(v *viper.Viper, requireToken bool) (cli.Config, error) {
	token, err := cliconfig.ResolveSecret(v.GetString("token"), v.GetString("token_command"))
	if err != nil {
		return cli.Config{}, fmt.Errorf("token_command: %w", err)
	}

	consumerToken, err := cliconfig.ResolveSecret(v.GetString("consumer_token"), v.GetString("consumer_token_command"))
	if err != nil {
		return cli.Config{}, fmt.Errorf("consumer_token_command: %w", err)
	}

	cfg := cli.Config{
		Server:        v.GetString("server"),
		Token:         token,
		ConsumerToken: consumerToken,
		Caller:        v.GetString("caller"),
		RequireToken:  requireToken,
	}

	if err := cfg.Validate(); err != nil {
		return cli.Config{}, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func configFilePath() (string, error) {
	path, err := cliconfig.Path("hush-hush-cli")
	if err != nil {
		return "", fmt.Errorf("resolve hush-hush-cli config path: %w", err)
	}

	return path, nil
}

// newInitCmd writes a starter config file populated with the same
// defaults the tool would otherwise fall back to, ready to edit
// (rules/cli.md).
func newInitCmd(v *viper.Viper, resolvePath func() (string, error)) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a starter config file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := resolvePath()
			if err != nil {
				return err
			}

			if cliconfig.Exists(path) && !force {
				return fmt.Errorf("%s: %w", path, errConfigAlreadyExists)
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes && term.IsTerminal(int(os.Stdin.Fd())) {
				return runInteractiveInit(v, cmd, path, bufio.NewScanner(cmd.InOrStdin()), term.ReadPassword)
			}

			return writeStarterConfig(cmd, path)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing config file")

	return cmd
}

func writeStarterConfig(cmd *cobra.Command, path string) error {
	if err := cliconfig.WriteStarter(path); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	return reportWrote(cmd, path)
}

// runInteractiveInit is the value-by-value flow behind init's own RunE and
// the pre-command nudge (maybeOfferInit): the prompting, persistence and
// writing live in cliconfig.WriteInteractive, seeded here with whatever
// viper already resolved. sc and readPassword are threaded through for the
// reasons its doc comment gives.
func runInteractiveInit(v *viper.Viper, cmd *cobra.Command, path string, sc *bufio.Scanner, readPassword cliconfig.PasswordReader) error {
	if err := cliconfig.WriteInteractive(path, sc, cmd.OutOrStdout(), int(os.Stdin.Fd()), readPassword, currentValues(v)); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	return reportWrote(cmd, path)
}

// currentValues seeds the prompts' defaults with whatever viper resolved.
func currentValues(v *viper.Viper) cliconfig.Values {
	return cliconfig.Values{
		Server:     v.GetString("server"),
		Caller:     v.GetString("caller"),
		Recipients: v.GetString("recipients"),
	}
}

func reportWrote(cmd *cobra.Command, path string) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path); err != nil {
		return fmt.Errorf("write init confirmation: %w", err)
	}

	return nil
}

// maybeOfferInit wires cliconfig.OfferInit to this process's real terminal
// and viper, then reloads the config it may just have written. The
// decision logic lives in cliconfig, where the confirmed-interactively
// branch is testable: go test's own stdin is never a TTY, so only the
// term.IsTerminal call below can't be exercised through root.Execute().
func maybeOfferInit(v *viper.Viper, resolvePath func() (string, error), cmd *cobra.Command) error {
	yes, _ := cmd.Flags().GetBool("yes")

	var path string

	wrote, err := cliconfig.OfferInit(cliconfig.OfferOptions{
		ResolvePath: func() (string, error) {
			var err error

			path, err = resolvePath()

			return path, err
		},
		In:           cmd.InOrStdin(),
		Out:          cmd.OutOrStdout(),
		Err:          cmd.ErrOrStderr(),
		FD:           int(os.Stdin.Fd()),
		ReadPassword: term.ReadPassword,
		Yes:          yes,
		Interactive:  term.IsTerminal(int(os.Stdin.Fd())),
		Current:      currentValues(v),
	})
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	if !wrote {
		return nil
	}

	if err := reportWrote(cmd, path); err != nil {
		return err
	}

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read newly written config: %w", err)
	}

	return nil
}
