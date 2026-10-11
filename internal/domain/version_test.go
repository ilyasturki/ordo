package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersionName(t *testing.T) {
	for name, want := range map[string]Semver{
		"v0.9.0":                  {0, 9, 0},
		"v0.1.0 - app polish":     {0, 1, 0},
		"v1.3.0 file diff viewer": {1, 3, 0},
		"v0.1.6b":                 {0, 1, 6},
		" v0.0.11 ":               {0, 0, 11},
	} {
		got, ok := ParseVersionName(name)
		assert.True(t, ok, name)
		assert.Equal(t, want, got, name)
	}
	for _, name := range []string{"v0", "v0.N.0", "next", "FUTURE", "0.9.0", "V0.9.0", "version v0.9.0"} {
		_, ok := ParseVersionName(name)
		assert.False(t, ok, name)
	}
}

func TestSubCategoryIsNeverAVersion(t *testing.T) {
	c := Category{ID: "c", Name: "v0.2.0", ParentID: "p"}
	assert.False(t, c.IsVersion())
	assert.ErrorIs(t, c.Ship(), ErrNotVersion)
}

func TestShipAndUnship(t *testing.T) {
	c := Category{ID: "c", Name: "v0.9.0"}
	require.NoError(t, c.Ship())
	assert.True(t, c.Shipped())
	assert.NotEmpty(t, c.ReleasedAt)
	assert.ErrorIs(t, c.Ship(), ErrAlreadyShipped)

	c.Name = "old stuff"
	assert.True(t, c.Shipped(), "a rename keeps the ship date")
	require.NoError(t, c.Unship())
	assert.False(t, c.Shipped())
	assert.ErrorIs(t, c.Unship(), ErrNotShipped)
	assert.ErrorIs(t, c.Ship(), ErrNotVersion)
}

func TestIsInboxName(t *testing.T) {
	for _, name := range []string{"next", "Next", "NEXT", "next 2", " next 12 "} {
		assert.True(t, IsInboxName(name), name)
	}
	for _, name := range []string{"NEXT REWORK", "next2", "nexts", "v0.9.0", "next 2b"} {
		assert.False(t, IsInboxName(name), name)
	}
}

func projectNamed(names ...string) Project {
	p := Project{}
	for i, n := range names {
		p.Categories = append(p.Categories, Category{ID: string(rune('a' + i)), Name: n})
	}
	return p
}

func TestInboxIndexSkipsSubCategories(t *testing.T) {
	p := projectNamed("v0.2.0", "next", "next 2")
	p.Categories[1].ParentID = "a"
	assert.Equal(t, 2, p.InboxIndex())
	none := projectNamed("todo")
	assert.Equal(t, -1, none.InboxIndex())
}

func TestNextVersionSlot(t *testing.T) {
	newestFirst := projectNamed("next 2", "v0.9.0", "v0.8.6", "v0.8.5")
	name, at, ok := newestFirst.NextVersionSlot()
	require.True(t, ok)
	assert.Equal(t, "v0.9.1", name)
	assert.Equal(t, 1, at, "above the newest version, below the inbox")

	oldestFirst := projectNamed("v0.0.1", "v0.1.0 - app polish", "v0.2.0", "MARKETING + SEO", "v1.0.0")
	name, at, ok = oldestFirst.NextVersionSlot()
	require.True(t, ok)
	assert.Equal(t, "v1.0.1", name)
	assert.Equal(t, 5, at, "after every category")

	kilorep := projectNamed("v0.2.0", "next", "v0.1.13", "v0.1.12")
	name, at, _ = kilorep.NextVersionSlot()
	assert.Equal(t, "v0.2.1", name)
	assert.Equal(t, 0, at)

	single := projectNamed("FUTURE", "v0.1.0")
	name, at, _ = single.NextVersionSlot()
	assert.Equal(t, "v0.1.1", name)
	assert.Equal(t, 1, at, "one version reads as newest first")

	flat := projectNamed("todo", "other")
	_, _, ok = flat.NextVersionSlot()
	assert.False(t, ok)
}
