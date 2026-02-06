package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/chris/shy/internal/summary"
)

// Styles
var (
	headerStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	focusDotStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	blurDotStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	normalStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	countStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	separatorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusBarStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

const marginX = 2

func (m *Model) renderView() string {
	var b strings.Builder

	width := m.width
	if width == 0 {
		width = 80
	}

	// Content width excludes left and right margins
	contentWidth := width - 2*marginX
	if contentWidth < 20 {
		contentWidth = 20
	}
	margin := strings.Repeat(" ", marginX)

	// Header
	b.WriteString(margin + m.renderHeader())
	b.WriteString("\n")
	b.WriteString(margin + separatorStyle.Render(strings.Repeat("=", contentWidth)))
	b.WriteString("\n\n")

	// Context list
	if len(m.contexts) == 0 {
		b.WriteString(margin + "No commands found\n")
	} else {
		// Calculate the max count width for alignment
		maxCount := 0
		for _, ctx := range m.contexts {
			if ctx.CommandCount > maxCount {
				maxCount = ctx.CommandCount
			}
		}
		countWidth := countColumnWidth(maxCount)

		for i, ctx := range m.contexts {
			b.WriteString(margin + m.renderContextItem(ctx, i == m.selectedIdx, contentWidth, countWidth))
			b.WriteString("\n")
		}
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(margin + separatorStyle.Render(strings.Repeat("─", contentWidth)))
	b.WriteString("\n")
	b.WriteString(margin + m.renderStatusBar())

	return b.String()
}

func (m *Model) renderHeader() string {
	dot := focusDotStyle.Render("●")
	if !m.focused {
		dot = blurDotStyle.Render("○")
	}

	dateStr := m.currentDate.Format("2006-01-02")
	dayName := m.currentDate.Format("Monday")
	relativeStr := m.relativeDateString()

	if relativeStr != "" {
		return headerStyle.Render("Work Summary") + " " + dot + " " + headerStyle.Render(fmt.Sprintf("%s %s (%s)", dayName, dateStr, relativeStr))
	}
	return headerStyle.Render("Work Summary") + " " + dot + " " + headerStyle.Render(fmt.Sprintf("%s %s", dayName, dateStr))
}

func (m *Model) relativeDateString() string {
	now := m.now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	current := time.Date(m.currentDate.Year(), m.currentDate.Month(), m.currentDate.Day(), 0, 0, 0, 0, time.Local)

	diff := today.Sub(current)
	days := int(diff.Hours() / 24)

	switch days {
	case 0:
		return "Today"
	case 1:
		return "Yesterday"
	default:
		return ""
	}
}

func (m *Model) renderContextItem(ctx ContextItem, selected bool, width int, countWidth int) string {
	// Format context name
	name := formatContextName(ctx.Key, ctx.Branch)

	// Format command count
	countText := fmt.Sprintf("%d commands", ctx.CommandCount)
	if ctx.CommandCount == 1 {
		countText = "1 command "
	}

	// Right-align: "  " prefix + name + padding + countText
	// prefix is 2 chars ("  " or "▶ ")
	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	// Available space for name: width - prefix(2) - gap(2) - countWidth
	gap := 2
	nameMaxWidth := width - len(prefix) - gap - countWidth
	if nameMaxWidth < 10 {
		nameMaxWidth = 10
	}

	// Truncate name if too long
	name = truncateWithEllipsis(name, nameMaxWidth)

	// Build the line with right-aligned count
	padding := width - ansi.StringWidth(prefix) - ansi.StringWidth(name) - ansi.StringWidth(countText)
	if padding < 1 {
		padding = 1
	}

	line := prefix + name + strings.Repeat(" ", padding) + countText

	if selected {
		return selectedStyle.Render(line)
	}
	return normalStyle.Render(line)
}

// countColumnWidth returns the width of the count column for alignment
func countColumnWidth(maxCount int) int {
	// "N commands" where N is the max count
	return len(fmt.Sprintf("%d commands", maxCount))
}

// truncateWithEllipsis truncates a string to maxWidth, adding … if truncated
func truncateWithEllipsis(s string, maxWidth int) string {
	if ansi.StringWidth(s) <= maxWidth {
		return s
	}
	// Truncate to maxWidth-1 to leave room for …
	truncated := ansi.Truncate(s, maxWidth-1, "")
	return truncated + "…"
}

func formatContextName(key summary.ContextKey, branch summary.BranchKey) string {
	dir := formatDir(key.WorkingDir)

	if key.GitRepo != "" && branch != summary.NoBranch {
		return fmt.Sprintf("%s:%s", dir, string(branch))
	}
	return dir
}

// formatDir converts a path for display. Uses ~ for home subdirectories,
// but keeps the full path when it is exactly the home directory.
func formatDir(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return path
	}
	return summary.TildePath(path)
}

func (m *Model) renderStatusBar() string {
	return statusBarStyle.Render("[j/k] Select  [Enter] Drill in  [h/l] Time  [t] Today  [y] Yesterday  [q] Quit")
}
