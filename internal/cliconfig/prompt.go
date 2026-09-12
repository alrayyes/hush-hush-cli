package cliconfig

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// PromptValue asks label (showing current as the default) on out, reading
// the answer from sc, and returns current unchanged on a bare Enter (or
// EOF) - the same "path of least resistance" shape as Confirm.
func PromptValue(sc *bufio.Scanner, out io.Writer, label, current string) (string, error) {
	if _, err := fmt.Fprintf(out, "%s [%s]: ", label, current); err != nil {
		return "", fmt.Errorf("write prompt: %w", err)
	}

	if !sc.Scan() {
		return current, nil
	}

	if line := strings.TrimSpace(sc.Text()); line != "" {
		return line, nil
	}

	return current, nil
}

// PasswordReader reads one line of masked input from a real terminal file
// descriptor - golang.org/x/term.ReadPassword's own signature, taken as a
// parameter so a caller not attached to a real TTY (a test) can fake one.
type PasswordReader func(fd int) ([]byte, error)

// PromptSecret asks label on out, then reads one masked line via
// readPassword against fd - production code passes term.ReadPassword and
// int(os.Stdin.Fd()); a test passes a fake reader and an arbitrary fd.
func PromptSecret(out io.Writer, label string, fd int, readPassword PasswordReader) (string, error) {
	if _, err := fmt.Fprintf(out, "%s: ", label); err != nil {
		return "", fmt.Errorf("write prompt: %w", err)
	}

	b, err := readPassword(fd)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", label, err)
	}

	// term.ReadPassword doesn't echo the Enter that ended input, so the
	// next line of output would otherwise run on straight after the
	// masked prompt.
	if _, err := fmt.Fprintln(out); err != nil {
		return "", fmt.Errorf("write newline: %w", err)
	}

	return strings.TrimSpace(string(b)), nil
}

// PersistChoice is how init should persist a credential field the user
// just typed - rules/cli.md's "secrets get a command option, not just a
// value", offered as an explicit menu rather than assumed.
type PersistChoice int

const (
	// PersistSkip means don't persist the value at all - it's used for
	// this invocation only (or, from PromptConfig's own perspective,
	// never entered in the first place).
	PersistSkip PersistChoice = iota
	// PersistKeyring means store the value in the OS credential store and
	// write the generated "hush-hush-cli config keyring-get <field>" as
	// the field's _command.
	PersistKeyring
	// PersistCommand means write a retrieval command the user supplied
	// themselves as the field's _command.
	PersistCommand
	// PersistLiteral means write the value itself into the config file.
	PersistLiteral
)

// PromptPersistence offers the four-way persistence choice for a
// credential field named by field (used only in the prompt text), reading
// from sc. An unrecognised or empty answer is PersistSkip - the same
// least-surprise default as Confirm and PromptValue: silence never opts
// a secret into being written to disk.
func PromptPersistence(sc *bufio.Scanner, out io.Writer, field string) (PersistChoice, string, error) {
	_, err := fmt.Fprintf(out, "Save %s for next time?\n"+
		"  1) OS keyring\n"+
		"  2) a command that retrieves it\n"+
		"  3) the literal value in the config file\n"+
		"  4) don't save it\n"+
		"choice [4]: ", field)
	if err != nil {
		return PersistSkip, "", fmt.Errorf("write prompt: %w", err)
	}

	if !sc.Scan() {
		return PersistSkip, "", nil
	}

	switch strings.TrimSpace(sc.Text()) {
	case "1":
		return PersistKeyring, "", nil
	case "2":
		if _, err := fmt.Fprint(out, "command: "); err != nil {
			return PersistSkip, "", fmt.Errorf("write prompt: %w", err)
		}

		if !sc.Scan() {
			return PersistSkip, "", nil
		}

		return PersistCommand, strings.TrimSpace(sc.Text()), nil
	case "3":
		return PersistLiteral, "", nil
	default:
		return PersistSkip, "", nil
	}
}

// Values is the plain (non-credential) connection settings PromptConfig
// shows as defaults, and the same three fields it returns unchanged.
type Values struct {
	Server     string
	Caller     string
	Recipients string
}

// CredentialAnswer is what PromptConfig collected for one credential
// field: the typed value (empty if the prompt was left blank, in which
// case Choice is always PersistSkip and Extra is meaningless) and how the
// caller should persist it.
type CredentialAnswer struct {
	Value  string
	Choice PersistChoice
	// Extra is the user-supplied retrieval command when Choice is
	// PersistCommand, empty otherwise.
	Extra string
}

// PromptResult is everything an interactive init run collected.
type PromptResult struct {
	Server     string
	Caller     string
	Recipients string
	Token      CredentialAnswer
	Identity   CredentialAnswer
}

// PromptConfig walks every connection setting in turn - server, token,
// caller, recipients, identity (design.md) - prompting on out and reading
// from in, showing current's values as defaults for the three plain
// fields. It creates exactly one *bufio.Scanner over in and threads it
// through every plain-value and persistence-choice prompt: see Confirm's
// doc comment for why a fresh Scanner per call would be wrong here.
func PromptConfig(in io.Reader, out io.Writer, fd int, readPassword PasswordReader, current Values) (PromptResult, error) {
	sc := bufio.NewScanner(in)

	server, err := PromptValue(sc, out, "Server URL", current.Server)
	if err != nil {
		return PromptResult{}, err
	}

	token, err := promptCredential(sc, out, "Token", fd, readPassword)
	if err != nil {
		return PromptResult{}, err
	}

	caller, err := PromptValue(sc, out, "Caller identity (optional)", current.Caller)
	if err != nil {
		return PromptResult{}, err
	}

	recipients, err := PromptValue(sc, out, "Recipients (comma-separated age public keys, optional)", current.Recipients)
	if err != nil {
		return PromptResult{}, err
	}

	identity, err := promptCredential(sc, out, "Identity (comma-separated age private keys)", fd, readPassword)
	if err != nil {
		return PromptResult{}, err
	}

	return PromptResult{
		Server:     server,
		Caller:     caller,
		Recipients: recipients,
		Token:      token,
		Identity:   identity,
	}, nil
}

// promptCredential reads one credential field's value, then - only when
// something was actually typed - offers the persistence choice. Leaving
// the value blank skips both the choice prompt and persistence itself:
// there's nothing to persist.
func promptCredential(sc *bufio.Scanner, out io.Writer, label string, fd int, readPassword PasswordReader) (CredentialAnswer, error) {
	value, err := PromptSecret(out, label, fd, readPassword)
	if err != nil {
		return CredentialAnswer{}, err
	}

	if value == "" {
		return CredentialAnswer{Choice: PersistSkip}, nil
	}

	choice, extra, err := PromptPersistence(sc, out, label)
	if err != nil {
		return CredentialAnswer{}, err
	}

	return CredentialAnswer{Value: value, Choice: choice, Extra: extra}, nil
}
