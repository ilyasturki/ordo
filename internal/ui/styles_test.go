package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagMarks_DiamondDrawsAsItsHuesDot(t *testing.T) {
	for _, hue := range []string{"green", "blue", "magenta", "cyan"} {
		dot, ok := tagColor(hue)
		assert.True(t, ok, hue)
		diamond, ok := tagColor(hue + "-diamond")
		assert.True(t, ok, hue+"-diamond")
		assert.Equal(t, dot, diamond, hue)
		assert.Equal(t, "● ", TagSegmentText(hue))
		assert.Equal(t, "● ", TagSegmentText(hue+"-diamond"))
	}
	assert.False(t, TagDotVisible("red-diamond"))
	assert.Equal(t, "", TagSegmentText(""))
}
