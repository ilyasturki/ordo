package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pinnedCategories specs are nestedCategories' with a leading "*" for a pinned one.
func pinnedCategories(specs ...string) []Category {
	cats := make([]Category, len(specs))
	for i, s := range specs {
		pinned := s != "" && s[0] == '*'
		if pinned {
			s = s[1:]
		}
		cats[i] = nestedCategories(s)[0]
		cats[i].Pinned = pinned
	}
	return cats
}

func pinShape(cats []Category) []string {
	out := categoryShape(cats)
	for i, c := range cats {
		if c.Pinned {
			out[i] = "*" + out[i]
		}
	}
	return out
}

func TestSettlePinnedPutsPinnedFirstAmongSiblings(t *testing.T) {
	cases := []struct {
		name     string
		in, want []string
	}{
		{"already settled", []string{"*a", "*b", "c"}, []string{"*a", "*b", "c"}},
		{"a newly pinned one ends the pinned run", []string{"*a", "b", "*c", "d"}, []string{"*a", "*c", "b", "d"}},
		{"a newly unpinned one starts the rest", []string{"a", "*b", "c"}, []string{"*b", "a", "c"}},
		{"a pinned parent carries its children", []string{"a", "b", "*c", "x<c", "y<c"}, []string{"*c", "x<c", "y<c", "a", "b"}},
		{"a pinned child goes first under its parent", []string{"a", "x<a", "*y<a", "b"}, []string{"a", "*y<a", "x<a", "b"}},
		{"a pinned child stays under its parent", []string{"a", "x<a", "*y<a", "*b"}, []string{"*b", "a", "*y<a", "x<a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cats := pinnedCategories(tc.in...)
			SettlePinned(cats)
			assert.Equal(t, tc.want, pinShape(cats))
		})
	}
}

func TestNormalizeNestingSettlesPins(t *testing.T) {
	p := Project{Categories: pinnedCategories("a", "*b", "x<b", "c", "*y<c")}
	NormalizeNesting(&p)
	assert.Equal(t, []string{"*b", "x<b", "a", "c", "*y<c"}, pinShape(p.Categories))
}

func TestReverseCategoriesReversesEachRun(t *testing.T) {
	p := Project{Categories: pinnedCategories("*a", "*b", "x<b", "c", "d")}
	p.ReverseCategories()
	assert.Equal(t, []string{"*b", "x<b", "*a", "d", "c"}, pinShape(p.Categories))
	p.ReverseCategories()
	assert.Equal(t, []string{"*a", "*b", "x<b", "c", "d"}, pinShape(p.Categories))
}

func TestNestingKeepsPinsFirst(t *testing.T) {
	t.Run("a pinned category nested goes first under its new parent", func(t *testing.T) {
		p := Project{Categories: pinnedCategories("*a", "x<a", "*b")}
		require.NoError(t, p.IndentCategories(2, 3))
		assert.Equal(t, []string{"*a", "*b<a", "x<a"}, pinShape(p.Categories))
	})
	t.Run("an unpinned category un-nested from a pinned parent lands after the pinned run", func(t *testing.T) {
		p := Project{Categories: pinnedCategories("*a", "x<a", "*b", "c")}
		at, err := p.OutdentCategories(1, 2)
		require.NoError(t, err)
		assert.Equal(t, []string{"*a", "*b", "x", "c"}, pinShape(p.Categories))
		assert.Equal(t, 2, at)
	})
}

func TestMergeKeepsPinsFirst(t *testing.T) {
	p := Project{Categories: pinnedCategories("a", "*x<a", "b", "y<b")}
	require.NoError(t, p.MergeCategory(0, 2))
	assert.Equal(t, []string{"b", "*x<b", "y<b"}, pinShape(p.Categories))
}
