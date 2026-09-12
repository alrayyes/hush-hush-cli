// Command hush-hush-cli is the client every hush-hush consumer speaks
// through - the writer's only interface to the service, and the same
// binary any consumer (CI job, deploy script) runs to fetch and decrypt a
// value locally. See CLAUDE.md and openspec/changes/secrets-object-store/
// for the design.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/alrayyes/hush-hush-cli/internal/cliconfig"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

// version is stamped in at build time by goreleaser, from the tag.
var version = "dev"

// errConfigAlreadyExists is a sentinel rather than a plain fmt.Errorf: a
// fixed condition (a file is already there), not a message built from
// per-call detail - the path itself is per-call detail, so it's wrapped
// in rather than folded into the message.
var errConfigAlreadyExists = errors.New("config file already exists (use --force to overwrite)")

// configEnvVars are every HUSH_HUSH_* variable a command reads - the
// persistent flags below plus recipients/identity, which are bound
// per-subcommand rather than on root. Used only to decide whether the
// tool is already configured through the environment, not to read a
// value.
var configEnvVars = []string{
	"HUSH_HUSH_SERVER", "HUSH_HUSH_TOKEN", "HUSH_HUSH_TOKEN_COMMAND", "HUSH_HUSH_CALLER",
	"HUSH_HUSH_RECIPIENTS", "HUSH_HUSH_IDENTITY",
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newRootCmd wires the persistent, env-overridable connection config
// (server URL, bearer token, caller identity) shared by every subcommand.
// HUSH_HUSH_SERVER, HUSH_HUSH_TOKEN, and HUSH_HUSH_CALLER override their
// matching flags - a CI job supplies these through its own secret storage,
// with no bespoke wrapper or Action (the cli spec's "runs unmodified
// inside CI" requirement). A config file at configPath() sits below both:
// rules/cli.md's flags > environment > config file > defaults.
func newRootCmd() *cobra.Command {
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
				return maybeOfferInit(cmd)
			}
		},
	}

	root.PersistentFlags().String("server", "http://localhost:8080", "hush-hush server URL")
	root.PersistentFlags().String("token", "", "write-path bearer token")
	root.PersistentFlags().String("token-command", "", "command whose trimmed stdout is the write-path bearer token (wins over --token if both are set)")
	root.PersistentFlags().String("caller", "", "self-presented identity recorded in the audit log")
	root.PersistentFlags().BoolP("yes", "y", false, "write a starter config with no prompt, if none exists")

	for _, name := range []string{"server", "token", "caller"} {
		_ = viper.BindPFlag(name, root.PersistentFlags().Lookup(name))
	}

	_ = viper.BindPFlag("token_command", root.PersistentFlags().Lookup("token-command"))

	viper.SetEnvPrefix("hush_hush")
	viper.AutomaticEnv()

	if path, err := configFilePath(); err == nil {
		viper.SetConfigFile(path)
		viper.SetConfigType("yaml")
		_ = viper.ReadInConfig() // no config file yet is not an error
	}

	root.AddCommand(newInitCmd())
	root.AddCommand(newInjectCmd())
	root.AddCommand(newGetCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newDeleteCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newManCmd(root))

	return root
}

// config resolves the CLI's connection settings, running --token-command/
// HUSH_HUSH_TOKEN_COMMAND if set - rules/cli.md's "secrets get a command
// option, not just a value", so a token can come from `pass`, an
// age-encrypted file, or a keyring CLI instead of sitting in the config
// file as plaintext. The command wins over a literal --token/token if both
// are set: whoever configured the command form did it on purpose.
//
// requireToken is set by inject/update/delete and left false by get: a
// missing token then fails Validate() here, before any request reaches
// the server, rather than surfacing as a bare 401 from deep inside the
// SDK.
func config(requireToken bool) (cli.Config, error) {
	token, err := cliconfig.ResolveSecret(viper.GetString("token"), viper.GetString("token_command"))
	if err != nil {
		return cli.Config{}, fmt.Errorf("token_command: %w", err)
	}

	cfg := cli.Config{
		Server:       viper.GetString("server"),
		Token:        token,
		Caller:       viper.GetString("caller"),
		RequireToken: requireToken,
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
func newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a starter config file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := configFilePath()
			if err != nil {
				return err
			}

			if cliconfig.Exists(path) && !force {
				return fmt.Errorf("%s: %w", path, errConfigAlreadyExists)
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes && term.IsTerminal(int(os.Stdin.Fd())) {
				return runInteractiveInit(cmd, path, bufio.NewScanner(cmd.InOrStdin()), term.ReadPassword)
			}

			return writeStarterConfig(cmd, path)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing config file")

	return cmd
}

const starterConfig = `# hush-hush-cli config file. Flags and HUSH_HUSH_* environment variables
# both override these - see README.md#configuration.
server: http://localhost:8080
token: ""
# token_command runs a command and uses its trimmed stdout as the token
# instead - it wins over the literal value above if both are set.
# token_command: "pass show hush-hush/write-token"
caller: ""
recipients: ""
identity: ""
# identity_command runs a command and uses its trimmed stdout as the
# identity instead - it wins over the literal value above if both are set.
# identity_command: "pass show hush-hush/identity-key"
`

func writeStarterConfig(cmd *cobra.Command, path string) error {
	if err := os.WriteFile(path, []byte(starterConfig), 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path); err != nil {
		return fmt.Errorf("write init confirmation: %w", err)
	}

	return nil
}

// runInteractiveInit is the value-by-value flow behind init's own RunE and
// the pre-command nudge (maybeOfferInit): prompt for every connection
// setting via cliconfig.PromptConfig, apply each credential field's chosen
// persistence, and write the result. sc and readPassword are threaded
// through as parameters rather than built from cmd.InOrStdin()/
// term.ReadPassword directly: maybeOfferInit needs to hand this the exact
// same scanner its own Confirm prompt just read from (see Confirm's doc
// comment for why a fresh one over the same reader would lose input), and
// a test needs to fake a TTY that doesn't exist in CI. init's own RunE
// builds a fresh scanner and passes term.ReadPassword itself.
func runInteractiveInit(cmd *cobra.Command, path string, sc *bufio.Scanner, readPassword cliconfig.PasswordReader) error {
	current := cliconfig.Values{
		Server:     viper.GetString("server"),
		Caller:     viper.GetString("caller"),
		Recipients: viper.GetString("recipients"),
	}

	result, err := cliconfig.PromptConfig(sc, cmd.OutOrStdout(), int(os.Stdin.Fd()), readPassword, current)
	if err != nil {
		return fmt.Errorf("prompt for config: %w", err)
	}

	token, err := persistCredential(result.Token, "token")
	if err != nil {
		return err
	}

	identity, err := persistCredential(result.Identity, "identity")
	if err != nil {
		return err
	}

	content := renderConfig(result.Server, token, result.Caller, result.Recipients, identity)

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path); err != nil {
		return fmt.Errorf("write init confirmation: %w", err)
	}

	return nil
}

// renderedSecret is a credential field's config-file representation after
// its persistence choice has been applied: at most one of Literal/Command
// is non-empty (PersistSkip leaves both empty, matching an unanswered
// field).
type renderedSecret struct {
	Literal string
	Command string
}

// persistCredential turns one prompted credential answer into its
// config-file representation, storing the value in the OS keyring first
// when that's the chosen persistence.
func persistCredential(answer cliconfig.CredentialAnswer, field string) (renderedSecret, error) {
	switch answer.Choice {
	case cliconfig.PersistKeyring:
		if err := keyring.Set(keyringService, field, answer.Value); err != nil {
			return renderedSecret{}, fmt.Errorf("store %s in keyring: %w", field, err)
		}

		return renderedSecret{Command: "hush-hush-cli config keyring-get " + field}, nil
	case cliconfig.PersistCommand:
		return renderedSecret{Command: answer.Extra}, nil
	case cliconfig.PersistLiteral:
		return renderedSecret{Literal: answer.Value}, nil
	default: // cliconfig.PersistSkip
		return renderedSecret{}, nil
	}
}

// renderConfig builds the YAML an interactive init writes. Every value is
// double-quoted via strconv.Quote regardless of content - simpler and
// safer than deciding case by case which values need it, at the cost of
// looking less like starterConfig's own hand-written, selectively-quoted
// style; that constant is untouched and still what --yes/no-TTY writes.
func renderConfig(server string, token renderedSecret, caller, recipients string, identity renderedSecret) string {
	var b strings.Builder

	b.WriteString("# hush-hush-cli config file. Flags and HUSH_HUSH_* environment variables\n")
	b.WriteString("# both override these - see README.md#configuration.\n")
	fmt.Fprintf(&b, "server: %s\n", strconv.Quote(server))
	fmt.Fprintf(&b, "token: %s\n", strconv.Quote(token.Literal))
	b.WriteString("# token_command runs a command and uses its trimmed stdout as the token\n")
	b.WriteString("# instead - it wins over the literal value above if both are set.\n")

	if token.Command != "" {
		fmt.Fprintf(&b, "token_command: %s\n", strconv.Quote(token.Command))
	} else {
		b.WriteString("# token_command: \"pass show hush-hush/write-token\"\n")
	}

	fmt.Fprintf(&b, "caller: %s\n", strconv.Quote(caller))
	fmt.Fprintf(&b, "recipients: %s\n", strconv.Quote(recipients))
	fmt.Fprintf(&b, "identity: %s\n", strconv.Quote(identity.Literal))
	b.WriteString("# identity_command runs a command and uses its trimmed stdout as the\n")
	b.WriteString("# identity instead - it wins over the literal value above if both are set.\n")

	if identity.Command != "" {
		fmt.Fprintf(&b, "identity_command: %s\n", strconv.Quote(identity.Command))
	} else {
		b.WriteString("# identity_command: \"pass show hush-hush/identity-key\"\n")
	}

	return b.String()
}

// maybeOfferInit is rules/cli.md's "a run with no config file and no
// relevant environment variable set offers to run init right there":
// skipped entirely once a config file exists or the environment already
// configures the tool, and never blocks a non-interactive run (no TTY)
// on a prompt nothing will ever answer.
//
// anyEnvSet is checked before ever resolving a path: ShouldWriteStarter
// always skips once it's true, and resolving one has a real side effect
// (creating the parent directory) that can fail on its own - a CI job
// that already sets every HUSH_HUSH_* variable, exactly the case
// cli.md's own "runs unmodified inside CI" requirement targets, must
// never be blocked by a nudge it was never going to act on anyway.
//
// The confirmed-interactively branch (yes is false, sc is non-nil) has no
// direct test through root.Execute(): go test's own stdin is never a TTY,
// so term.IsTerminal below is always false in CI, same limitation
// TestRunInteractiveInit*'s own doc comment already notes for init itself.
// What it delegates to is fully covered there instead - runInteractiveInit
// end-to-end, and Confirm/PromptConfig sharing one scanner as
// TestConfirmSharesAScannerWithLaterPrompts (cliconfig_test.go) - so this
// function's own job, wiring the two together with the right scanner, is
// what's left untested at the cobra level, not the behavior itself.
func maybeOfferInit(cmd *cobra.Command) error {
	anyEnvSet := anyConfigEnvVarSet()
	if anyEnvSet {
		return nil
	}

	path, err := configFilePath()
	if err != nil {
		// Advisory only: a run this environment doesn't already
		// configure still has to work even where the config path
		// itself can't be resolved or created.
		return nil //nolint:nilerr // advisory only, error already explained above
	}

	exists := cliconfig.Exists(path)
	yes, _ := cmd.Flags().GetBool("yes")
	interactive := term.IsTerminal(int(os.Stdin.Fd()))

	sc, confirmed := confirmInitNudge(cmd, exists, anyEnvSet, yes, interactive)

	if !cliconfig.ShouldWriteStarter(exists, anyEnvSet, yes, interactive, confirmed) {
		return printUnconfiguredNudge(cmd, exists, anyEnvSet, interactive)
	}

	if err := writeConfig(cmd, path, yes, sc); err != nil {
		return err
	}

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read newly written config: %w", err)
	}

	return nil
}

// confirmInitNudge asks "set one up now?" only when every gating
// condition for offering it holds, and returns the scanner it read the
// answer from alongside that answer: a caller that goes on to run the
// interactive flow on a yes needs that exact scanner, not a fresh one
// over the same reader - see Confirm's doc comment for why.
func confirmInitNudge(cmd *cobra.Command, exists, anyEnvSet, yes, interactive bool) (*bufio.Scanner, bool) {
	if yes || !interactive || exists || anyEnvSet {
		return nil, false
	}

	sc := bufio.NewScanner(cmd.InOrStdin())
	confirmed := cliconfig.Confirm(sc, cmd.OutOrStdout(), "No config file found. Set one up now?")

	return sc, confirmed
}

// printUnconfiguredNudge is the stderr fallback for the one case
// ShouldWriteStarter leaves nothing written for: fully non-interactive
// and entirely unconfigured, where there was never a prompt to answer.
func printUnconfiguredNudge(cmd *cobra.Command, exists, anyEnvSet, interactive bool) error {
	if exists || anyEnvSet || interactive {
		return nil
	}

	if _, err := fmt.Fprintf(cmd.ErrOrStderr(),
		"no config file and no HUSH_HUSH_* environment variables set - running on defaults (`hush-hush-cli init` writes a starter config)\n",
	); err != nil {
		return fmt.Errorf("write config nudge: %w", err)
	}

	return nil
}

// writeConfig is what ShouldWriteStarter having said yes actually does:
// --yes writes the same blank template a non-interactive init would; the
// confirmed-interactively path (sc is non-nil whenever that's how
// ShouldWriteStarter came to true) runs the full interactive flow.
func writeConfig(cmd *cobra.Command, path string, yes bool, sc *bufio.Scanner) error {
	if yes {
		return writeStarterConfig(cmd, path)
	}

	return runInteractiveInit(cmd, path, sc, term.ReadPassword)
}

func anyConfigEnvVarSet() bool {
	for _, name := range configEnvVars {
		if _, ok := os.LookupEnv(name); ok {
			return true
		}
	}

	return false
}
