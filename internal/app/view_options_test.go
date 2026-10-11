package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/config"
)

func currentOptionValue(t *testing.T, m *model, i int) string {
	t.Helper()
	return optionSpecs[i].current(m.deps.CfgManager.Get())
}

func TestCycleOption_WrapsBothWays(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Options.selectedOption = 1 // Priority Color

	require.Equal(t, config.PriorityColorFull, currentOptionValue(t, m, 1))
	for _, want := range []string{config.PriorityColorIcon, config.PriorityColorNone, config.PriorityColorFull} {
		m.cycleSelectedOption(1)
		assert.Equal(t, want, currentOptionValue(t, m, 1))
	}
	m.cycleSelectedOption(-1)
	assert.Equal(t, config.PriorityColorNone, currentOptionValue(t, m, 1), "h is l's inverse, and wraps too")
}

func TestOptionSpecs_EveryValueRoundTrips(t *testing.T) {
	m := newTestModel(t, sampleProject())
	for _, spec := range optionSpecs {
		keys := make([]string, len(spec.values))
		for j, v := range spec.values {
			keys[j] = v.key
		}
		// Or the dialog would highlight nothing and h/l would land on the first.
		assert.Containsf(t, keys, spec.current(m.deps.CfgManager.Get()), "option %s", spec.name)

		for _, v := range spec.values {
			spec.apply(m, v.key)
			assert.Equalf(t, v.key, spec.current(m.deps.CfgManager.Get()), "option %s value %s", spec.name, v.key)
		}
	}
}

func TestResetSelectedOption_RestoresDefault(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Options.selectedOption = 3 // Descriptions, whose apply also flips the screen flag
	m.cycleSelectedOption(1)
	require.Equal(t, "on", currentOptionValue(t, m, 3))
	require.True(t, m.ui.Screen.ExpandDescriptions)

	m.resetSelectedOption()
	assert.Equal(t, "off", currentOptionValue(t, m, 3))
	assert.False(t, m.ui.Screen.ExpandDescriptions, "reset goes through apply, not just the config")
}

func TestOptionRow_MarksChangedValues(t *testing.T) {
	m := newTestModel(t, sampleProject())
	spec := optionSpecs[1] // Priority Color
	row := func() string { return m.optionRow(spec, m.deps.CfgManager.Get(), false, 60) }

	assert.NotContains(t, row(), changedMark)
	spec.apply(m, config.PriorityColorNone)
	assert.Contains(t, row(), changedMark)
	m.ui.Options.selectedOption = 1
	m.resetSelectedOption()
	assert.NotContains(t, row(), changedMark)
}
