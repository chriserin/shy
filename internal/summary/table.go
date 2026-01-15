package summary

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// FormatTable formats context summaries as a table
func FormatTable(summaries []ContextSummary, date string, isYesterday bool) string {
	if len(summaries) == 0 {
		return formatEmptyState(date, isYesterday)
	}

	// Sort summaries by duration (desc), then command count (desc), then directory (asc)
	sortSummaries(summaries)

	var sb strings.Builder

	// Header
	sb.WriteString(formatTableHeader(date, isYesterday))
	sb.WriteString("\n\n")

	// Calculate column widths
	widths := calculateColumnWidths(summaries)

	// Table header row
	sb.WriteString(formatColumnHeaders(widths))
	sb.WriteString("\n")

	// Table rows
	for _, summary := range summaries {
		sb.WriteString(formatTableRow(summary, widths))
		sb.WriteString("\n")
	}

	// Summary statistics
	sb.WriteString("\n")
	sb.WriteString(formatSummaryStats(summaries))

	return sb.String()
}

func formatEmptyState(date string, isYesterday bool) string {
	header := formatTableHeader(date, isYesterday)
	return header + "\n\nNo commands found for this date."
}

func formatTableHeader(date string, isYesterday bool) string {
	var title string
	if isYesterday {
		title = fmt.Sprintf("Yesterday's Work Summary - %s", date)
	} else {
		title = fmt.Sprintf("Work Summary - %s", date)
	}
	separator := strings.Repeat("=", len(title))
	return title + "\n" + separator
}

func formatColumnHeaders(widths columnWidths) string {
	return fmt.Sprintf("%-*s  %-*s  %*s  %-*s  %s",
		widths.directory, "Directory",
		widths.branch, "Branch",
		widths.commands, "Commands",
		widths.timeSpan, "Time Span",
		"Duration")
}

func formatTableRow(summary ContextSummary, widths columnWidths) string {
	dir := normalizeDirectory(summary.WorkingDir)
	branch := summary.BranchDisplay()
	commands := strconv.Itoa(summary.CommandCount)
	timeSpan := summary.FormatTimeSpan()
	duration := summary.FormatDuration()

	return fmt.Sprintf("%-*s  %-*s  %*s  %-*s  %s",
		widths.directory, dir,
		widths.branch, branch,
		widths.commands, commands,
		widths.timeSpan, timeSpan,
		duration)
}

func formatSummaryStats(summaries []ContextSummary) string {
	totalCommands := 0
	for _, summary := range summaries {
		totalCommands += summary.CommandCount
	}

	plural := ""
	if len(summaries) != 1 {
		plural = "s"
	}

	return fmt.Sprintf("Total: %d commands across %d context%s",
		totalCommands, len(summaries), plural)
}

type columnWidths struct {
	directory int
	branch    int
	commands  int
	timeSpan  int
}

func calculateColumnWidths(summaries []ContextSummary) columnWidths {
	widths := columnWidths{
		directory: len("Directory"),
		branch:    len("Branch"),
		commands:  len("Commands"),
		timeSpan:  len("Time Span"),
	}

	for _, summary := range summaries {
		dir := normalizeDirectory(summary.WorkingDir)
		if len(dir) > widths.directory {
			widths.directory = len(dir)
		}

		branch := summary.BranchDisplay()
		if len(branch) > widths.branch {
			widths.branch = len(branch)
		}

		commandStr := strconv.Itoa(summary.CommandCount)
		if len(commandStr) > widths.commands {
			widths.commands = len(commandStr)
		}

		timeSpan := summary.FormatTimeSpan()
		if len(timeSpan) > widths.timeSpan {
			widths.timeSpan = len(timeSpan)
		}
	}

	return widths
}

func normalizeDirectory(path string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	// Replace home directory with ~
	if strings.HasPrefix(path, homeDir+"/") {
		return "~" + path[len(homeDir):]
	}
	if path == homeDir {
		return "~"
	}

	return path
}

func sortSummaries(summaries []ContextSummary) {
	sort.Slice(summaries, func(i, j int) bool {
		// Primary sort: duration (descending)
		durationI := summaries[i].Duration()
		durationJ := summaries[j].Duration()
		if durationI != durationJ {
			return durationI > durationJ
		}

		// Secondary sort: command count (descending)
		if summaries[i].CommandCount != summaries[j].CommandCount {
			return summaries[i].CommandCount > summaries[j].CommandCount
		}

		// Tertiary sort: directory (ascending)
		dirI := normalizeDirectory(summaries[i].WorkingDir)
		dirJ := normalizeDirectory(summaries[j].WorkingDir)
		if dirI != dirJ {
			return dirI < dirJ
		}

		// Final sort: branch (ascending)
		branchI := summaries[i].BranchDisplay()
		branchJ := summaries[j].BranchDisplay()
		return branchI < branchJ
	})
}
