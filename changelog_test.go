package ordo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChangelogDropsLinkReferences(t *testing.T) {
	assert.Contains(t, changelogFile, "[Unreleased]: https://")
	assert.NotContains(t, Changelog, "/compare/")
	assert.NotContains(t, Changelog, "[0.1.1]: ")
	assert.True(t, strings.HasSuffix(Changelog, "and a Nix flake.\n"), Changelog[max(len(Changelog)-80, 0):])
}
