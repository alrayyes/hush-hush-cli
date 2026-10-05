// Command hush-hush-cli is the client every hush-hush consumer speaks
// through - the writer's only interface to the service, and the same
// binary any consumer (CI job, deploy script) runs to fetch and decrypt a
// value locally. See CLAUDE.md and openspec/changes/secrets-object-store/
// for the design. The commands live in internal/cmd; this file only starts
// them.
package main

import (
	"fmt"
	"os"

	"github.com/alrayyes/hush-hush-cli/internal/cmd"
)

// version is stamped in at build time by goreleaser, from the tag.
var version = "dev"

func main() {
	if err := cmd.NewRootCmd(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
