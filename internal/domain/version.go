package domain

import (
	"cmp"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var versionNamePattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)`)

type Semver struct {
	Major, Minor, Patch int
}

func (v Semver) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Semver) Compare(o Semver) int {
	return cmp.Or(cmp.Compare(v.Major, o.Major), cmp.Compare(v.Minor, o.Minor), cmp.Compare(v.Patch, o.Patch))
}

func (v Semver) NextPatch() Semver {
	return Semver{v.Major, v.Minor, v.Patch + 1}
}

// ParseVersionName reads the semver a name starts with; a label may follow ("v0.1.0 - app polish").
func ParseVersionName(name string) (Semver, bool) {
	m := versionNamePattern.FindStringSubmatch(strings.TrimSpace(name))
	if m == nil {
		return Semver{}, false
	}
	var parts [3]int
	for i := range parts {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return Semver{}, false
		}
		parts[i] = n
	}
	return Semver{parts[0], parts[1], parts[2]}, true
}

// Version is the semver of a top-level category named after one.
func (c *Category) Version() (Semver, bool) {
	if c.ParentID != "" {
		return Semver{}, false
	}
	return ParseVersionName(c.Name)
}

func (c *Category) IsVersion() bool {
	_, ok := c.Version()
	return ok
}

func (p *Project) VersionIndexes() []int {
	var out []int
	for i := range p.Categories {
		if p.Categories[i].IsVersion() {
			out = append(out, i)
		}
	}
	return out
}

// NewestOnTop reports a project listing its versions newest first; one version or none reads as newest first.
func (p *Project) NewestOnTop() bool {
	idxs := p.VersionIndexes()
	if len(idxs) < 2 {
		return true
	}
	first, _ := p.Categories[idxs[0]].Version()
	last, _ := p.Categories[idxs[len(idxs)-1]].Version()
	return first.Compare(last) >= 0
}

func (p *Project) NewestVersion() int {
	best := -1
	var bestV Semver
	for _, i := range p.VersionIndexes() {
		v, _ := p.Categories[i].Version()
		if best < 0 || v.Compare(bestV) > 0 {
			best, bestV = i, v
		}
	}
	return best
}
