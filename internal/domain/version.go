package domain

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

type VersionPart int

const (
	// PatchPart is the last part a version has: the patch of v1.2.3, the minor of v1.2.
	PatchPart VersionPart = iota
	MinorPart
	MajorPart
)

var (
	ErrNoVersion   = errors.New("no version in this name")
	ErrVersionZero = errors.New("the version is already at 0")

	versionPattern = regexp.MustCompile(`\bv?(\d+)\.(\d+)(?:\.(\d+))?`)
)

// versionSpans are the byte ranges of the first version's major, minor and patch; patch is nil on a two-part version.
func versionSpans(name string) [][]int {
	for _, m := range versionPattern.FindAllStringSubmatchIndex(name, -1) {
		if rest := name[m[1]:]; len(rest) >= 2 && rest[0] == '.' && rest[1] >= '0' && rest[1] <= '9' {
			continue
		}
		spans := [][]int{{m[2], m[3]}, {m[4], m[5]}}
		if m[6] >= 0 {
			spans = append(spans, []int{m[6], m[7]})
		}
		return spans
	}
	return nil
}

func HasVersion(name string) bool {
	return versionSpans(name) != nil
}

// BumpVersion adds delta to one part of the first version in name, zeroing the parts after it; only the digits change.
func BumpVersion(name string, part VersionPart, delta int) (string, error) {
	spans := versionSpans(name)
	if spans == nil {
		return "", ErrNoVersion
	}
	at := len(spans) - 1
	switch part {
	case MinorPart:
		at = 1
	case MajorPart:
		at = 0
	}
	values := make([]int, len(spans))
	for i, s := range spans {
		n, err := strconv.Atoi(name[s[0]:s[1]])
		if err != nil {
			return "", ErrNoVersion
		}
		values[i] = n
	}
	if values[at]+delta < 0 {
		return "", ErrVersionZero
	}
	values[at] += delta
	for i := at + 1; i < len(values); i++ {
		values[i] = 0
	}
	var b strings.Builder
	end := 0
	for i, s := range spans {
		b.WriteString(name[end:s[0]])
		b.WriteString(formatLike(name[s[0]:s[1]], values[i]))
		end = s[1]
	}
	b.WriteString(name[end:])
	return b.String(), nil
}

// formatLike keeps a zero-padded part padded ("05" + 1 = "06").
func formatLike(orig string, n int) string {
	s := strconv.Itoa(n)
	if len(orig) > 1 && orig[0] == '0' && len(s) < len(orig) {
		s = strings.Repeat("0", len(orig)-len(s)) + s
	}
	return s
}
