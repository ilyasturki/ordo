package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
}
