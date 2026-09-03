// Command hush-hush-cli is the client for the hush-hush secrets object
// store - the writer's and every consumer's own interface to it.
//
// This is a placeholder: the real implementation (inject/get/update/delete,
// config, init) still lives in github.com/alrayyes/hush-hush and moves here
// as tracked migration tasks - see CONTRIBUTING.md and
// openspec/changes/split-cli-into-own-repo/.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is stamped in at build time by goreleaser, from the tag.
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "hush-hush-cli",
		Short:   "Client for the hush-hush secrets object store",
		Version: version,
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
