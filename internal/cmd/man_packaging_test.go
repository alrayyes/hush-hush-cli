package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra/doc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEveryManPageIsPackaged keeps .goreleaser.yml's explicit page lists in
// step with the command tree. They're explicit rather than a glob because
// cobra's own completion-* pages are left out on purpose, so a new command
// would otherwise ship without a page and nothing would notice.
func TestEveryManPageIsPackaged(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	header := &doc.GenManHeader{Title: "HUSH-HUSH-CLI", Section: "1"}
	require.NoError(t, doc.GenManTree(newBareRoot(t), header, dir))

	pages, err := filepath.Glob(filepath.Join(dir, "*.1"))
	require.NoError(t, err)
	require.NotEmpty(t, pages)

	config, err := os.ReadFile("../../.goreleaser.yml")
	require.NoError(t, err)

	for _, path := range pages {
		page := filepath.Base(path)
		if strings.HasPrefix(page, "hush-hush-cli-completion") {
			continue
		}

		assert.Contains(t, string(config), "src: manpages/"+page, "archives and nfpms both read it from here")
		assert.Contains(t, string(config), "dst: man1/"+page, "missing from the archive")
		assert.Contains(t, string(config), "dst: /usr/share/man/man1/"+page, "missing from the .deb/.rpm")
	}
}
