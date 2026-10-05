package cliconfig

import (
	"bufio"
	"fmt"
	"io"
)

// OfferOptions is everything OfferInit needs from its caller, so none of it
// has to know about cobra, viper or a real terminal.
type OfferOptions struct {
	// ResolvePath finds (and may create the directory for) the config file
	// path. It's a callback so OfferInit can skip it entirely when the
	// environment already configures the tool: resolving a path has a real
	// side effect that can fail on a CI runner with no writable home.
	ResolvePath func() (string, error)

	In  io.Reader
	Out io.Writer
	Err io.Writer

	// FD and ReadPassword read a secret without echo.
	FD           int
	ReadPassword PasswordReader

	// Yes is the --yes flag. Interactive is whether stdin is a terminal a
	// prompt can use.
	Yes         bool
	Interactive bool

	// Current seeds the prompts' defaults.
	Current Values
}

// OfferInit is cli.md's "a run with no config file and no relevant
// environment variable set offers to run init right there": skipped once a
// config file exists or the environment already configures the tool, and
// never blocks a non-interactive run on a prompt nothing will answer. It
// reports whether it wrote a config file. A path that can't be resolved is
// advisory only: a run this environment doesn't configure still has to work.
func OfferInit(o OfferOptions) (bool, error) {
	anyEnvSet := AnyEnvVarSet()
	if anyEnvSet {
		return false, nil
	}

	path, err := o.ResolvePath()
	if err != nil {
		return false, nil //nolint:nilerr // advisory only, see above
	}

	exists := Exists(path)

	// One scanner for the confirm and the prompts after it: a second one
	// over the same reader would lose whatever the first had buffered.
	var (
		sc        *bufio.Scanner
		confirmed bool
	)

	if !o.Yes && o.Interactive && !exists {
		sc = bufio.NewScanner(o.In)
		confirmed = Confirm(sc, o.Out, "No config file found. Set one up now?")
	}

	if !ShouldWriteStarter(exists, anyEnvSet, o.Yes, o.Interactive, confirmed) {
		return false, printUnconfiguredNudge(o.Err, exists, o.Interactive)
	}

	if o.Yes {
		return true, WriteStarter(path)
	}

	return true, WriteInteractive(path, sc, o.Out, o.FD, o.ReadPassword, o.Current)
}

// printUnconfiguredNudge is the stderr fallback for the one case
// ShouldWriteStarter leaves nothing written for: fully non-interactive and
// entirely unconfigured, where there was never a prompt to answer.
func printUnconfiguredNudge(w io.Writer, exists, interactive bool) error {
	if exists || interactive {
		return nil
	}

	if _, err := fmt.Fprintf(w,
		"no config file and no HUSH_HUSH_* environment variables set - running on defaults (`hush-hush-cli init` writes a starter config)\n",
	); err != nil {
		return fmt.Errorf("write config nudge: %w", err)
	}

	return nil
}
