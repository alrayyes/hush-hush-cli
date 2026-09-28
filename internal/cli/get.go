package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/alrayyes/hush-hush-cli/internal/client"
	"github.com/alrayyes/hush-hush-cli/internal/seal"
)

// errNoReadCredentialMsg names both ways to authorize a read, since Get
// (unlike inject/update/delete) accepts either one - Validate's
// RequireToken check can't catch this ahead of time, since neither field
// is individually required, so this only ever fires after the server has
// already rejected the request (readError wraps client.ErrUnauthorized
// around it, unlike cli.go's errTokenRequired, which fires before any
// request is made).
const errNoReadCredentialMsg = "no token configured (--token/--consumer-token, " +
	"HUSH_HUSH_TOKEN/HUSH_HUSH_CONSUMER_TOKEN, or run `hush-hush-cli init` for a write token)"

// Get fetches id's ciphertext and decrypts it locally with whichever of
// identities matches - the CLI never writes an assembled file or applies
// consumer-side file-shape logic (design.md).
//
// It builds its own client rather than using Config.newClient(): a write
// Token, when set, always takes priority over ConsumerToken - a write
// token already authorizes reads, and there's exactly one Authorization
// header per request, so the two are never sent together (design.md's
// "Decisions").
func Get(ctx context.Context, cfg Config, id string, identities []string) ([]byte, error) {
	credential := cfg.Token
	if credential == "" {
		credential = cfg.ConsumerToken
	}

	cl, err := client.New(cfg.Server, credential)
	if err != nil {
		return nil, fmt.Errorf("build client: %w", err)
	}

	cl.Caller = cfg.Caller

	sealed, err := cl.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch object: %w", readError(err, cfg))
	}

	plaintext, err := seal.Unseal(sealed, identities)
	if err != nil {
		return nil, fmt.Errorf("decrypt object: %w", err)
	}

	return plaintext, nil
}

// readError turns a bare client.ErrUnauthorized into one naming which
// credential was missing or rejected, per this change's cli-config delta
// ("Actionable error on a rejected read"). Any other error passes through
// unchanged.
func readError(err error, cfg Config) error {
	if !errors.Is(err, client.ErrUnauthorized) {
		return err
	}

	switch {
	case cfg.Token == "" && cfg.ConsumerToken == "":
		return fmt.Errorf("%s: %w", errNoReadCredentialMsg, client.ErrUnauthorized)
	case cfg.Token != "":
		return fmt.Errorf("write token rejected: %w", client.ErrUnauthorized)
	default:
		return fmt.Errorf("consumer token rejected: %w", client.ErrUnauthorized)
	}
}
