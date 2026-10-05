package render

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/alrayyes/hush-hush-cli/internal/client"
)

// ConsumerTokenWithValue prints a freshly minted or rotated token, value included, as a one-row table or raw JSON.
func ConsumerTokenWithValue(out io.Writer, token client.ConsumerTokenWithValue, asJSON bool) error {
	if asJSON {
		if err := json.NewEncoder(out).Encode(token); err != nil {
			return fmt.Errorf("write consumer token as json: %w", err)
		}

		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintln(w, "ID\tCONSUMER\tDESCRIPTION\tEXPIRES\tTOKEN"); err != nil {
		return fmt.Errorf("write consumer token header: %w", err)
	}

	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		token.ID, token.Consumer, token.Description, token.ExpiresAt.Local().Format(time.DateTime), token.Value); err != nil {
		return fmt.Errorf("write consumer token row: %w", err)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush consumer token table: %w", err)
	}

	return nil
}

// ConsumerTokensJSON prints tokens as one JSON array, `[]` for none.
func ConsumerTokensJSON(out io.Writer, tokens []client.ConsumerToken) error {
	if tokens == nil {
		tokens = []client.ConsumerToken{}
	}

	if err := json.NewEncoder(out).Encode(tokens); err != nil {
		return fmt.Errorf("write consumer tokens as json: %w", err)
	}

	return nil
}

// ConsumerTokensTable prints tokens as an aligned table with their status.
func ConsumerTokensTable(out io.Writer, tokens []client.ConsumerToken) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintln(w, "ID\tCONSUMER\tDESCRIPTION\tEXPIRES\tSTATUS"); err != nil {
		return fmt.Errorf("write token list header: %w", err)
	}

	for _, t := range tokens {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			t.ID, t.Consumer, t.Description, t.ExpiresAt.Local().Format(time.DateTime), t.Status); err != nil {
			return fmt.Errorf("write token list row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush token list table: %w", err)
	}

	return nil
}
