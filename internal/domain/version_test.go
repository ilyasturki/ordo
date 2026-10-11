package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBumpVersion(t *testing.T) {
	cases := []struct {
		name  string
		part  VersionPart
		delta int
		want  string
	}{
		{"v0.4.1", PatchPart, 1, "v0.4.2"},
		{"v0.4.2", PatchPart, -1, "v0.4.1"},
		{"v0.4.2 - polish", MinorPart, 1, "v0.5.0 - polish"},
		{"v0.4.2", MajorPart, 1, "v1.0.0"},
		{"v0.1.9", PatchPart, 1, "v0.1.10"},
		{"v0.1.10", PatchPart, -1, "v0.1.9"},
		{"v0.1.6b", PatchPart, 1, "v0.1.7b"},
		{"v1.3.0 file diff viewer", PatchPart, 1, "v1.3.1 file diff viewer"},
		{"v1.0.0 - PUBLIC RELEASE + MARKETING", MinorPart, 1, "v1.1.0 - PUBLIC RELEASE + MARKETING"},
		{"release v1.2 soon", PatchPart, 1, "release v1.3 soon"},
		{"v1.2", MinorPart, 1, "v1.3"},
		{"v1.2", MajorPart, 1, "v2.0"},
		{"1.4.0", PatchPart, 1, "1.4.1"},
		{"(v2.0.0)", PatchPart, 1, "(v2.0.1)"},
		{"v1.2.3 then v4.5.6", PatchPart, 1, "v1.2.4 then v4.5.6"},
		{"v1.2.3.4 or v2.0.0", PatchPart, 1, "v1.2.3.4 or v2.0.1"},
		{"v2026.05", PatchPart, 1, "v2026.06"},
	}
	for _, tc := range cases {
		got, err := BumpVersion(tc.name, tc.part, tc.delta)
		require.NoError(t, err, tc.name)
		assert.Equal(t, tc.want, got, tc.name)
	}
}

func TestBumpVersionRefusals(t *testing.T) {
	for _, name := range []string{"MARKETING + SEO", "v0", "v0.N.0", "25/03/2026", "Week 13", "v1.2.3.4", "next", "x1.2", "v1_2.3", ""} {
		_, err := BumpVersion(name, PatchPart, 1)
		assert.ErrorIs(t, err, ErrNoVersion, name)
		assert.False(t, HasVersion(name), name)
	}
	_, err := BumpVersion("v0.4.0", PatchPart, -1)
	assert.ErrorIs(t, err, ErrVersionZero)
	_, err = BumpVersion("v1.0", PatchPart, -1)
	assert.ErrorIs(t, err, ErrVersionZero, "a two-part version takes its minor as the last part")
}
