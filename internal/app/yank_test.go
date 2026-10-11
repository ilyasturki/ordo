package app

import (
	"reflect"
	"testing"

	"github.com/ilyasturki/phasionary/internal/app/components"
	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/domain"
)

func entityValues(items []components.YankItem) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Value
	}
	return out
}

func TestExtractEntities(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", []string{""}, nil},
		{"nothing notable", []string{"refactor the auth flow"}, nil},
		{"uuid", []string{"resource 550e8400-e29b-41d4-a716-446655440000 broke"},
			[]string{"550e8400-e29b-41d4-a716-446655440000"}},
		{"url", []string{"see https://example.com/x"}, []string{"https://example.com/x"}},
		{"bare number", []string{"bump to version 42"}, []string{"42"}},
		{"order uuid url number", []string{"PR 7 fixes 550e8400-e29b-41d4-a716-446655440000 see https://ex.com"},
			[]string{"550e8400-e29b-41d4-a716-446655440000", "https://ex.com", "7"}},
		{"digits inside uuid not reported", []string{"id 550e8400-e29b-41d4-a716-446655440000"},
			[]string{"550e8400-e29b-41d4-a716-446655440000"}},
		{"digits inside url not reported", []string{"open https://example.com/123"},
			[]string{"https://example.com/123"}},
		{"dedupe across texts", []string{"ticket 99", "still ticket 99"}, []string{"99"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := entityValues(extractEntities(tc.in...))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("extractEntities(%q) values = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestBuildYankItemsTask(t *testing.T) {
	project := domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{{
			ID:   "c1",
			Name: "Cat A",
			Tasks: []domain.Task{{
				ID:          "t1",
				Title:       "Fix 550e8400-e29b-41d4-a716-446655440000 in flow",
				Status:      domain.StatusTodo,
				Description: "details at https://example.com/issue",
			}},
		}},
	}
	m := newTestModel(t, project)
	m.ui.Selection.MoveTo(2) // 0=project, 1=Cat A header, 2=task

	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusTask {
		t.Fatalf("expected a focused task, got %+v ok=%v", pos, ok)
	}
	got := entityValues(m.buildYankItems(pos))
	want := []string{
		"Fix 550e8400-e29b-41d4-a716-446655440000 in flow",
		"details at https://example.com/issue",
		"550e8400-e29b-41d4-a716-446655440000",
		"https://example.com/issue",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildYankItems values = %#v, want %#v", got, want)
	}
}
