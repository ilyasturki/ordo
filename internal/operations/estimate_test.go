package operations_test

import (
	"testing"

	"github.com/ilyasturki/ordo/internal/operations"
)

func TestParseEstimate(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{
		{"empty is zero", "", 0, false},
		{"zero", "0", 0, false},
		{"plain minutes", "90", 90, false},
		{"hours", "2h", 120, false},
		{"fractional hours", "1.5h", 90, false},
		{"bare minutes suffix", "30m", 30, false},
		{"hours and minutes", "2h30m", 150, false},
		{"whitespace trimmed", "  2h  ", 120, false},
		{"uppercase", "2H", 120, false},
		{"negative rejected", "-5", 0, true},
		{"garbage rejected", "abc", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := operations.ParseEstimate(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseEstimate(%q) = %d, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseEstimate(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParseEstimate(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
