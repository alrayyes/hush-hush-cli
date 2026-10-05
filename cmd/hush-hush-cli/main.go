// Command hush-hush-cli is the client every hush-hush consumer speaks
// through - the writer's only interface to the service, and the same
// binary any consumer (CI job, deploy script) runs to fetch and decrypt a
// value locally. See CLAUDE.md and openspec/changes/secrets-object-store/
// for the design. The commands live in internal/cmd; this file only starts
// them.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alrayyes/hush-hush-cli/internal/cmd"
)

// version is stamped in at build time by goreleaser, from the tag.
var version = "dev"

// exitInterrupted is the shell convention (128 + SIGINT) for a run ended
// by Ctrl-C.
const exitInterrupted = 130

func main() {
	os.Exit(run())
}

// run is main without the os.Exit, so the signal handler's stop runs.
// Ctrl-C (or SIGTERM) cancels the context every command's requests carry,
// so a slow or hung server can be given up on at once.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := cmd.NewRootCmd(version).ExecuteContext(ctx)

	switch {
	case err == nil:
		return 0
	case errors.Is(err, context.Canceled) && ctx.Err() != nil:
		fmt.Fprintln(os.Stderr, "interrupted")

		return exitInterrupted
	default:
		fmt.Fprintln(os.Stderr, err)

		return 1
	}
}
