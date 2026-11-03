package ordo

import (
	_ "embed"
	"regexp"
	"strings"
)

//go:embed CHANGELOG.md
var changelogFile string

var linkReference = regexp.MustCompile(`(?m)^\[[^\]]+\]: .*\n?`)

// Changelog is CHANGELOG.md without the link-reference block that ends it.
var Changelog = strings.TrimRight(linkReference.ReplaceAllString(changelogFile, ""), "\n") + "\n"
