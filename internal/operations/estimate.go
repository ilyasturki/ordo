// Package operations holds the shared, pure task and category commands used by
// the CLI and web front doors. Every verb operates on an in-memory
// *domain.Project and performs no I/O; persistence stays with each front door
// (their WithProjectLocked wrappers). This keeps the commands trivially
// testable against project literals and gives the two surfaces one place to
// agree on validation and mutation.
package operations

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var timeEstimateRe = regexp.MustCompile(`^(?:(\d+(?:\.\d+)?)d)?(?:(\d+(?:\.\d+)?)h)?(?:(\d+)m?)?$`)

// ParseEstimate interprets a human estimate string as a whole number of
// minutes. It accepts "" (= 0), a plain minute count ("90"), days of 8 hours
// as domain.FormatEstimate prints them ("2d"), hours ("2h", "1.5h"), minutes
// ("30m"), or a combination ("1d4h", "2h30m"). It rejects negative
// and unparseable input — the union of the CLI's grammar and the web's guard.
func ParseEstimate(raw string) (int, error) {
	input := strings.TrimSpace(strings.ToLower(raw))
	if input == "" {
		return 0, nil
	}

	if mins, err := strconv.Atoi(input); err == nil {
		if mins < 0 {
			return 0, ErrNegativeEstimate
		}
		return mins, nil
	}

	m := timeEstimateRe.FindStringSubmatch(input)
	if m == nil {
		return 0, fmt.Errorf("invalid time estimate format: %s", raw)
	}

	var total float64
	if m[1] != "" {
		days, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, err
		}
		total += days * 8 * 60
	}
	if m[2] != "" {
		hours, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return 0, err
		}
		total += hours * 60
	}
	if m[3] != "" {
		mins, err := strconv.Atoi(m[3])
		if err != nil {
			return 0, err
		}
		total += float64(mins)
	}
	return int(total), nil
}
