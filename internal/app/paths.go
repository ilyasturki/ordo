package app

import (
	"regexp"
	"strings"
)

// pathPattern is the one path detector: gy's paths and the path under the
// caret. A path starts a word, or follows an opening delimiter, and ends at
// whitespace or markup, so a name with a space is cut there.
var pathPattern = regexp.MustCompile(`(?:^|[\s(\[{<"'\x60])((?:file://|~/|\.\.?/|/)[^\s<>"'\x60]*)`)

var pathPrefixes = []string{"file://", "~/", "../", "./", "/"}

// extractPaths lists the paths written in text, as typed, in order.
func extractPaths(text string) []string {
	var out []string
	for _, loc := range pathPattern.FindAllStringSubmatchIndex(text, -1) {
		p := strings.TrimRight(text[loc[2]:loc[3]], urlTrailingTrim)
		if !barePrefix(p) {
			out = append(out, p)
		}
	}
	return out
}

func barePrefix(p string) bool {
	for _, prefix := range pathPrefixes {
		if rest, ok := strings.CutPrefix(p, prefix); ok {
			return rest == ""
		}
	}
	return true
}
