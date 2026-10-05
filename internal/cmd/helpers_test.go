package cmd_test

import (
	"path/filepath"
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Tests in this package call t.Parallel() unless they touch process-wide
// state, which is any test that calls t.Setenv or keyring.MockInit. The
// remaining ones are on purpose:
//   - the environment layer itself (the "FromEnvironment..." tests,
//     config_test.go, init_test.go) sets HUSH_HUSH_* and XDG_CONFIG_HOME,
//   - the keyring tests swap the keyring backend, a package-level provider,
//   - tags_test.go's tagsEnv, and the tests that read HUSH_HUSH_RECIPIENTS or
//     HUSH_HUSH_IDENTITY, set those through the environment because
//     --recipients and --identity are per-subcommand flags newRoot doesn't
//     set.
//
// newRoot builds a root command pointed at server with token, through
// flags and a private config file rather than HUSH_HUSH_* and
// XDG_CONFIG_HOME. It touches no process-wide state, so a test that uses
// only it can call t.Parallel(). A test of the environment layer itself
// uses t.Setenv instead, and says so.
func newRoot(t *testing.T, server, token string) *cobra.Command {
	t.Helper()

	root := cmd.NewRootCmd("dev", cmd.WithConfigPath(filepath.Join(t.TempDir(), "config.yaml")))
	require.NoError(t, root.PersistentFlags().Set("server", server))
	require.NoError(t, root.PersistentFlags().Set("token", token))

	return root
}

// newBareRoot is newRoot without a server or token, for tests that never
// reach a request. It still takes a private config path: resolving the
// default one reloads the xdg package's global directories, which races
// with a parallel test doing the same.
func newBareRoot(t *testing.T) *cobra.Command {
	t.Helper()

	return cmd.NewRootCmd("dev", cmd.WithConfigPath(filepath.Join(t.TempDir(), "config.yaml")))
}
