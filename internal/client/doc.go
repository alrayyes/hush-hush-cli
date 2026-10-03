// Package client is the CLI's transport to the hush-hush server - a thin
// adapter over the generated hush-hush-go SDK, kept so internal/cli depends
// on this package's own sentinel errors and ObjectMetadata shape rather than
// the SDK's directly (design.md).
package client
