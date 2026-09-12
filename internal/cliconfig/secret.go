package cliconfig

import (
	"fmt"
	"os/exec"
	"strings"
)

// ResolveSecret is the shared implementation behind every "<field>"/
// "<field>_command" pair (token/token_command, identity/identity_command):
// rules/cli.md's "secrets get a command option, not just a value". command
// wins over literal if both are set - whoever configured the command form
// did it on purpose.
func ResolveSecret(literal, command string) (string, error) {
	if command == "" {
		return literal, nil
	}

	out, err := runSecretCommand(command)
	if err != nil {
		return "", fmt.Errorf("run: %w", err)
	}

	return out, nil
}

// runSecretCommand runs cmdStr through the shell (so a pipeline like `pass
// show <path>` works unmodified) and returns its trimmed stdout - exactly
// one trailing newline, not every trailing space, so a secret that
// genuinely ends in whitespace survives.
func runSecretCommand(cmdStr string) (string, error) {
	out, err := exec.Command("sh", "-c", cmdStr).Output() //nolint:gosec // cmdStr is operator-supplied config, not external input
	if err != nil {
		return "", fmt.Errorf("run: %w", err)
	}

	return strings.TrimSuffix(string(out), "\n"), nil
}
