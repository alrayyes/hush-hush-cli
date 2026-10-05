// Package render prints command results, as JSON or as an aligned table, to
// any io.Writer. It knows nothing about cobra: internal/cmd hands it
// cmd.OutOrStdout() and the data a command fetched.
package render
