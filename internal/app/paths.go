package app

import (
	"path/filepath"
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

// pathBeforeCaret is the path the text before the caret ends in, and the rune
// index it starts at.
func pathBeforeCaret(line []rune, caret int) (string, int, bool) {
	text := string(line[:caret])
	locs := pathPattern.FindAllStringSubmatchIndex(text, -1)
	if len(locs) == 0 {
		return "", 0, false
	}
	last := locs[len(locs)-1]
	if last[3] != len(text) {
		return "", 0, false
	}
	return text[last[2]:], len([]rune(text[:last[2]])), true
}

// resolvePath turns a typed path into one the filesystem takes; a relative
// one is against ordo's working directory.
func resolvePath(typed string) (string, bool) {
	if strings.HasPrefix(typed, "./") || strings.HasPrefix(typed, "../") {
		abs, err := filepath.Abs(typed)
		return abs, err == nil
	}
	p, ok := dropPath(typed)
	return filepath.Clean(p), ok
}
