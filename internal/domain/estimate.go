package domain

import "fmt"

// FormatEstimate counts a day as 8 hours, a work day.
func FormatEstimate(minutes int) string {
	if minutes == 0 {
		return ""
	}
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	if hours < 8 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 8
	return fmt.Sprintf("%dd", days)
}

func EstimateLabel(minutes int) string {
	if minutes == 0 {
		return "None"
	}
	if minutes < 60 {
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	hours := minutes / 60
	if hours < 8 {
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := hours / 8
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
