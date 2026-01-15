package summary

import (
	"strconv"
	"time"
)

// ContextSummary represents aggregated command data for a specific context
// A context is defined by the combination of working directory and git branch
type ContextSummary struct {
	WorkingDir   string
	GitBranch    *string // nil for non-git directories
	CommandCount int
	FirstTime    int64 // Unix timestamp
	LastTime     int64 // Unix timestamp
}

// Duration returns the time span between first and last command in seconds
func (c *ContextSummary) Duration() int64 {
	return c.LastTime - c.FirstTime
}

// FormatDuration returns a human-readable duration string
// Examples: "8h 12m", "45m", "30s", "0s"
func (c *ContextSummary) FormatDuration() string {
	duration := c.Duration()

	if duration == 0 {
		return "0s"
	}

	hours := duration / 3600
	minutes := (duration % 3600) / 60
	seconds := duration % 60

	if hours > 0 {
		if minutes > 0 {
			return formatHoursMinutes(hours, minutes)
		}
		return formatHours(hours)
	}

	if minutes > 0 {
		return formatMinutes(minutes)
	}

	return formatSeconds(seconds)
}

func formatHoursMinutes(hours, minutes int64) string {
	return formatWithSuffix(hours, "h") + " " + formatWithSuffix(minutes, "m")
}

func formatHours(hours int64) string {
	return formatWithSuffix(hours, "h")
}

func formatMinutes(minutes int64) string {
	return formatWithSuffix(minutes, "m")
}

func formatSeconds(seconds int64) string {
	return formatWithSuffix(seconds, "s")
}

func formatWithSuffix(value int64, suffix string) string {
	return strconv.FormatInt(value, 10) + suffix
}

// FormatTimeSpan returns the time range as "HH:MM - HH:MM"
func (c *ContextSummary) FormatTimeSpan() string {
	firstTime := time.Unix(c.FirstTime, 0)
	lastTime := time.Unix(c.LastTime, 0)

	return formatTime(firstTime) + " - " + formatTime(lastTime)
}

func formatTime(t time.Time) string {
	hour := t.Hour()
	minute := t.Minute()

	return formatTwoDigits(hour) + ":" + formatTwoDigits(minute)
}

func formatTwoDigits(n int) string {
	if n < 10 {
		return "0" + string(rune(n+'0'))
	}
	tens := n / 10
	ones := n % 10
	return string(rune(tens+'0')) + string(rune(ones+'0'))
}

// BranchDisplay returns the branch name or "-" for non-git directories
func (c *ContextSummary) BranchDisplay() string {
	if c.GitBranch == nil {
		return "-"
	}
	return *c.GitBranch
}
