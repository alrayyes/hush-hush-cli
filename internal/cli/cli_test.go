package cli_test

import (
	"testing"

	"github.com/alrayyes/hush-hush-cli/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		server  string
		wantErr string
	}{
		"valid":       {server: "http://localhost:8080", wantErr: ""},
		"empty":       {server: "", wantErr: "server: required"},
		"no scheme":   {server: "localhost:8080", wantErr: "not an absolute URL"},
		"scheme only": {server: "http://", wantErr: "not an absolute URL"},
		"unparseable": {server: "http://[::1", wantErr: "server:"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := cli.Config{Server: tc.server}.Validate()

			if tc.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestConfigValidateRequiresTokenWhenRequireTokenIsSet(t *testing.T) {
	t.Parallel()

	t.Run("missing token", func(t *testing.T) {
		t.Parallel()

		err := cli.Config{Server: "http://localhost:8080", RequireToken: true}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "token")
	})

	t.Run("token present", func(t *testing.T) {
		t.Parallel()

		err := cli.Config{Server: "http://localhost:8080", Token: "abc", RequireToken: true}.Validate()
		require.NoError(t, err)
	})

	t.Run("token not required", func(t *testing.T) {
		t.Parallel()

		err := cli.Config{Server: "http://localhost:8080"}.Validate()
		require.NoError(t, err)
	})
}
