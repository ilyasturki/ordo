package app

import (
	"path/filepath"
	"regexp"
	"strings"
)

// A space ends a path, so a name holding one is cut there.
var pathPattern = regexp.MustCompile(`(?:^|[\s(\[{<"'\x60])((?:file://|~/|\.\.?/|/)[^\s<>"'\x60]*)`)

var pathPrefixes = []string{"file://", "~/", "../", "./", "/"}

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

// A relative path resolves against ordo's working directory.
func resolvePath(typed string) (string, bool) {
	if strings.HasPrefix(typed, "./") || strings.HasPrefix(typed, "../") {
		abs, err := filepath.Abs(typed)
		return abs, err == nil
	}
	p, ok := dropPath(typed)
	return filepath.Clean(p), ok
}
