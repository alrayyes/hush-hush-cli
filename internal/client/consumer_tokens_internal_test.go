package client

import (
	"testing"
	"time"

	hushhush "github.com/alrayyes/hush-hush-go/v4"
	"github.com/stretchr/testify/assert"
)

// An older server sends neither status nor allowed_actions; the CLI has to
// work them out itself so the table never prints a blank.
func TestToConsumerTokenDerivesStateWhenServerOmitsIt(t *testing.T) {
	t.Parallel()

	now := time.Now()

	cases := map[string]struct {
		meta        hushhush.ConsumerTokenMetadata
		wantStatus  string
		wantActions []string
	}{
		"active":  {hushhush.ConsumerTokenMetadata{ExpiresAt: now.Add(time.Hour)}, "active", []string{"rotate", "revoke"}},
		"expired": {hushhush.ConsumerTokenMetadata{ExpiresAt: now.Add(-time.Hour)}, "expired", []string{"purge"}},
		"revoked": {hushhush.ConsumerTokenMetadata{ExpiresAt: now.Add(time.Hour), Revoked: true}, "revoked", []string{"purge"}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := toConsumerToken(tc.meta)
			assert.Equal(t, tc.wantStatus, got.Status)
			assert.Equal(t, tc.wantActions, got.AllowedActions)
		})
	}
}
