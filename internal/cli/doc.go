// Package cli implements every hush-hush CLI command as a plain function
// over Config, independent of cobra - the actual command definitions in
// cmd/hush-hush-cli are a thin parsing shell around these
// (rules/go.md: "Keep RunE a thin shell").
package cli
