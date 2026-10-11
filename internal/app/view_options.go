package app

import (
	"slices"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/ui"
)

type optionValue struct {
	key   string
	label string
}

type optionSpec struct {
	name    string
	detail  string
	values  []optionValue
	current func(cfg config.Config) string
	apply   func(m *model, key string)
}

var optionSpecs = []optionSpec{
	{
		name:    "Status Display",
		detail:  "the status column as [x] glyphs or as words",
		values:  []optionValue{{key: config.StatusDisplayIcons, label: "Icons"}, {key: config.StatusDisplayText, label: "Text"}},
		current: func(cfg config.Config) string { return cfg.StatusDisplay },
		apply: func(m *model, key string) {
			_ = m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.StatusDisplay = key })
			// Icons vs. text change the status column width, so cached row heights are stale.
			m.invalidateLayout()
		},
	},
	{
		name:   "Priority Color",
		detail: "color the whole title, the priority arrow alone, or nothing",
		values: []optionValue{
			{key: config.PriorityColorFull, label: "Full"},
			{key: config.PriorityColorIcon, label: "Icon"},
			{key: config.PriorityColorNone, label: "None"},
		},
		current: func(cfg config.Config) string { return cfg.PriorityColor },
		apply: func(m *model, key string) {
			_ = m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.PriorityColor = key })
		},
	},
	{
		name:    "Shortcut Bar",
		detail:  "the key hints along the bottom of the screen",
		values:  onOff,
		current: func(cfg config.Config) string { return onOffKey(cfg.ShowShortcutBar) },
		apply: func(m *model, key string) {
			_ = m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.ShowShortcutBar = key == "on" })
		},
	},
	{
		name:    "Descriptions",
		detail:  "whether descriptions start expanded; zd toggles them",
		values:  onOff,
		current: func(cfg config.Config) string { return onOffKey(cfg.ExpandDescriptionsByDefault) },
		apply: func(m *model, key string) {
			on := key == "on"
			_ = m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.ExpandDescriptionsByDefault = on })
			m.ui.Screen.ExpandDescriptions = on
			m.rebuildPositions()
		},
	},
	{
		name:    "Category Counts",
		detail:  "done / total after each top-level category name",
		values:  onOff,
		current: func(cfg config.Config) string { return onOffKey(cfg.ShowCategoryCounts) },
		apply: func(m *model, key string) {
			_ = m.deps.CfgManager.Update(func(cfg *config.Config) { cfg.ShowCategoryCounts = key == "on" })
			// Counts narrow the header name, so cached row heights are stale.
			m.invalidateLayout()
		},
	},
}

var onOff = []optionValue{{key: "on", label: "On"}, {key: "off", label: "Off"}}

func onOffKey(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

const changedMark = " •"

func (spec optionSpec) defaultValue() string { return spec.current(config.DefaultConfig()) }

func (m *model) resetSelectedOption() {
	spec := optionSpecs[m.ui.Options.selectedOption]
	spec.apply(m, spec.defaultValue())
}

func (m *model) cycleSelectedOption(delta int) {
	spec := optionSpecs[m.ui.Options.selectedOption]
	cur := spec.current(m.deps.CfgManager.Get())
	i := max(slices.IndexFunc(spec.values, func(v optionValue) bool { return v.key == cur }), 0)
	n := len(spec.values)
	spec.apply(m, spec.values[((i+delta)%n+n)%n].key)
}

func (m model) optionsView() string {
	cfg := m.deps.CfgManager.Get()
	width := m.dialogWidth()

	lines := []string{ui.DialogTitleStyle.Render("Options"), ""}
	for i, spec := range optionSpecs {
		lines = append(lines,
			m.optionRow(spec, cfg, i == m.ui.Options.selectedOption, width),
			ui.MutedStyle.Render("    "+spec.detail),
			"",
		)
	}
	lines = append(lines, ui.RenderHintsToWidth([]ui.Hint{
		{Key: "h/l", Label: "change"},
		{Key: "d", Label: "default"},
		{Key: "j/k", Label: "move"},
		{Key: strings.TrimSpace(changedMark), Label: "changed"},
		{Key: "q/esc", Label: "close"},
	}, width))
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}

func (m model) optionRow(spec optionSpec, cfg config.Config, focused bool, width int) string {
	base := lipgloss.NewStyle()
	if focused {
		base = ui.GetSelectedStyle(m.ui.Screen.WindowFocused)
	}

	cur := spec.current(cfg)
	values := make([]string, len(spec.values))
	for i, v := range spec.values {
		values[i] = base.Bold(v.key == cur || focused).Faint(v.key != cur).Render(v.label)
	}

	left := base.Render("  ") + ui.DialogKey(base, focused).Render(spec.name)
	if cur != spec.defaultValue() {
		left += base.Render(changedMark)
	}
	right := strings.Join(values, base.Render("  ")) + base.Render("  ")
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + base.Render(strings.Repeat(" ", gap)) + right
}
