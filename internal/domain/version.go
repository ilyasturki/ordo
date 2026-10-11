package domain

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrNotVersion      = errors.New("not a version: its name must start with vX.Y.Z")
	ErrAlreadyShipped  = errors.New("already shipped")
	ErrNotShipped      = errors.New("not shipped")
	versionNamePattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)`)
	inboxNamePattern   = regexp.MustCompile(`(?i)^next( \d+)?$`)
)

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

// Shipped follows ReleasedAt alone, so renaming a shipped version never reopens it.
func (c *Category) Shipped() bool {
	return c.ReleasedAt != ""
}

func (c *Category) Ship() error {
	if !c.IsVersion() {
		return ErrNotVersion
	}
	if c.Shipped() {
		return ErrAlreadyShipped
	}
	now := NowTimestamp()
	c.ReleasedAt, c.UpdatedAt = now, now
	return nil
}

func (c *Category) Unship() error {
	if !c.Shipped() {
		return ErrNotShipped
	}
	c.ReleasedAt, c.UpdatedAt = "", NowTimestamp()
	return nil
}

func IsInboxName(name string) bool {
	return inboxNamePattern.MatchString(strings.TrimSpace(name))
}

// InboxIndex is the first top-level category named next or "next N", or -1.
func (p *Project) InboxIndex() int {
	for i, c := range p.Categories {
		if c.ParentID == "" && IsInboxName(c.Name) {
			return i
		}
	}
	return -1
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

// NextVersionSlot proposes the patch after the newest version and where it goes:
// above the newest in a newest-first project, after every category otherwise.
func (p *Project) NextVersionSlot() (name string, index int, ok bool) {
	newest := p.NewestVersion()
	if newest < 0 {
		return "", 0, false
	}
	v, _ := p.Categories[newest].Version()
	if p.NewestOnTop() {
		return v.NextPatch().String(), newest, true
	}
	return v.NextPatch().String(), len(p.Categories), true
}
