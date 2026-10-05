// Package cmd is the cobra command tree behind hush-hush-cli: NewRootCmd
// wires the persistent flags, the config file and every subcommand, and
// cmd/hush-hush-cli only starts it. The commands stay thin parsing shells
// over internal/cli and internal/cliconfig.
package cmd
