package cmd

import (
	"errors"

	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/spf13/cobra"
)

var errIDAndConsumer = errors.New("--id and --consumer both pick a variant; give one")

// variantFlags is the pair of ways get, update, delete and used-by pick one
// variant of a name that has several.
type variantFlags struct {
	id       string
	consumer string
}

func (f *variantFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.id, "id", "", "UUID of one variant, for a name that has several (see list --json)")
	cmd.Flags().StringVar(&f.consumer, "consumer", "", "pick the variant used by this consumer, for a name that has several")
}

// validate refuses both selectors at once, before any request: the server
// would answer 400.
func (f variantFlags) validate() error {
	if f.id != "" && f.consumer != "" {
		return errIDAndConsumer
	}

	return nil
}

func (f variantFlags) callOptions() []client.CallOption {
	return []client.CallOption{client.WithVariant(f.id), client.WithConsumer(f.consumer)}
}
