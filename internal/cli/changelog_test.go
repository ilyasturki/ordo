package cli

import (
	"strings"
	"testing"

	"github.com/ilyasturki/ordo"
)

func TestChangelogPrintsTheEmbeddedFile(t *testing.T) {
	out, err := runCLI(t, t.TempDir(), "changelog")
	if err != nil {
		t.Fatal(err)
	}
	if out != ordo.Changelog {
		t.Fatalf("output differs from the embedded changelog (%d vs %d bytes)", len(out), len(ordo.Changelog))
	}
	if !strings.HasPrefix(out, "# Changelog\n") || strings.Contains(out, "/compare/") {
		t.Fatalf("unexpected output:\n%.200s", out)
	}
}
