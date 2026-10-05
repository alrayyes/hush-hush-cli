package cliconfig

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// WriteStarter writes the blank StarterConfig template to path, owner-only:
// the file can hold a literal credential once someone fills it in.
func WriteStarter(path string) error {
	if err := os.WriteFile(path, []byte(StarterConfig), 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// WriteInteractive is the value-by-value flow behind init and the
// pre-command nudge: prompt for every connection setting via PromptConfig,
// apply each credential field's chosen persistence, and write the result
// to path. sc and readPassword are parameters rather than built here so the
// nudge can hand over the exact scanner its own Confirm prompt just read
// from (see Confirm for why a fresh one would lose input), and a test can
// fake a TTY that doesn't exist in CI.
func WriteInteractive(path string, sc *bufio.Scanner, out io.Writer, fd int, readPassword PasswordReader, current Values) error {
	result, err := PromptConfig(sc, out, fd, readPassword, current)
	if err != nil {
		return fmt.Errorf("prompt for config: %w", err)
	}

	token, err := PersistCredential(result.Token, "token")
	if err != nil {
		return fmt.Errorf("persist token: %w", err)
	}

	identity, err := PersistCredential(result.Identity, "identity")
	if err != nil {
		return fmt.Errorf("persist identity: %w", err)
	}

	content := Render(result.Server, token, result.Caller, result.Recipients, identity)

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
