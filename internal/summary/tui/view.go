package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/chris/shy/internal/summary"
	"github.com/chris/shy/pkg/models"
)

// Styles
var (
	headerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	focusDotStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	blurDotStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	selectedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	normalStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	countStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	separatorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bucketLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
)

const marginX = 2

func (m *Model) renderView() string {
	switch m.viewState {
	case ContextDetailView:
		return m.renderDetailView()
	default:
		return m.renderSummaryView()
	}
}

func (m *Model) renderSummaryView() string {
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
	contentLines := 0
	if len(m.contexts) == 0 {
		b.WriteString(margin + "No commands found\n")
		contentLines = 1
	} else {
		// Calculate the max count width for alignment
		maxCount := 0
		for _, ctx := range m.contexts {
			c := filteredCommandCount(ctx.Commands, m.displayMode)
			if c > maxCount {
				maxCount = c
			}
		}
		countWidth := countColumnWidth(maxCount)

		for i, ctx := range m.contexts {
			b.WriteString(margin + m.renderContextItem(ctx, i == m.selectedIdx, contentWidth, countWidth))
			b.WriteString("\n")
		}
		contentLines = len(m.contexts)
	}

	// Pad to push footer to bottom
	// Fixed lines: header(1) + ===(1) + blank(1) + content + blank(1) + ---(1) + statusbar(1) = 6 + content
	if m.height > 0 {
		avail := m.height - 6
		if avail > contentLines {
			for i := 0; i < avail-contentLines; i++ {
				b.WriteString("\n")
			}
		}
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(margin + separatorStyle.Render(strings.Repeat("─", contentWidth)))
	b.WriteString("\n")
	b.WriteString(margin + m.renderStatusBar())

	return b.String()
}

func (m *Model) renderDetailView() string {
	var b strings.Builder

	width := m.width
	if width == 0 {
		width = 80
	}

	contentWidth := width - 2*marginX
	if contentWidth < 20 {
		contentWidth = 20
	}
	margin := strings.Repeat(" ", marginX)

	// Header (fixed)
	b.WriteString(margin + m.renderHeader())
	b.WriteString("\n")
	b.WriteString(margin + separatorStyle.Render(strings.Repeat("=", contentWidth)))
	b.WriteString("\n")

	contentLines := 0
	if len(m.detailCommands) == 0 {
		b.WriteString("\n" + margin + "No commands found\n")
		contentLines = 2 // blank + "No commands found"
	} else {
		// Build all body lines
		var bodyLines []string
		cmdIdx := 0
		for _, bucket := range m.detailBuckets {
			// Blank line before bucket
			bodyLines = append(bodyLines, "")
			// Bucket header
			label := bucketLabelStyle.Render(bucket.Label)
			dashWidth := contentWidth - 2 - ansi.StringWidth(bucket.Label) - 1
			if dashWidth < 2 {
				dashWidth = 2
			}
			bodyLines = append(bodyLines, margin+"  "+label+" "+separatorStyle.Render(strings.Repeat("─", dashWidth)))
			// Commands
			for _, cmd := range bucket.Commands {
				bodyLines = append(bodyLines, margin+m.renderDetailCommand(cmd, cmdIdx == m.detailCmdIdx))
				cmdIdx++
			}
		}

		// Apply viewport scrolling if height is set
		if m.height > 0 {
			avail := m.height - 5 // header(1) + ===(1) + blank(1) + ---footer(1) + statusbar(1)
			if avail < 1 {
				avail = 1
			}
			start := m.detailScrollOffset
			if start > len(bodyLines) {
				start = len(bodyLines)
			}
			end := start + avail
			if end > len(bodyLines) {
				end = len(bodyLines)
			}
			bodyLines = bodyLines[start:end]
		}

		for _, line := range bodyLines {
			b.WriteString(line)
			b.WriteString("\n")
		}
		contentLines = len(bodyLines)
	}

	// Pad to push footer to bottom
	// Fixed lines: header(1) + ===(1) + content + blank(1) + ---(1) + statusbar(1) = 5 + content
	if m.height > 0 {
		avail := m.height - 5
		if avail > contentLines {
			for i := 0; i < avail-contentLines; i++ {
				b.WriteString("\n")
			}
		}
	}

	// Footer (fixed)
	b.WriteString("\n")
	b.WriteString(margin + separatorStyle.Render(strings.Repeat("─", contentWidth)))
	b.WriteString("\n")
	b.WriteString(margin + m.renderStatusBar())

	return b.String()
}

func (m *Model) renderDetailCommand(cmd models.Command, selected bool) string {
	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	t := time.Unix(cmd.Timestamp, 0)
	minute := t.Format(":04")

	cmdText := cmd.CommandText
	if parts := strings.SplitN(cmdText, "\n", 2); len(parts) > 1 {
		cmdText = parts[0] + " ↵"
	}

	if m.displayMode == MultiMode {
		if count, ok := m.detailFrequencies[cmd.CommandText]; ok && count > 1 {
			cmdText = cmdText + fmt.Sprintf("  ⟳ %d", count)
		}
	}

	line := prefix + "  " + minute + "  " + cmdText

	if selected {
		return selectedStyle.Render(line)
	}
	return normalStyle.Render(line)
}

func (m *Model) renderHeader() string {
	dot := focusDotStyle.Render("●")
	if !m.focused {
		dot = blurDotStyle.Render("○")
	}

	title := "Work Summary"
	if m.viewState == ContextDetailView {
		title = formatContextName(m.detailContextKey, m.detailContextBranch)
	}

	dateStr := m.currentDate.Format("2006-01-02")
	dayName := m.currentDate.Format("Monday")
	relativeStr := m.relativeDateString()

	if relativeStr != "" {
		return headerStyle.Render(title) + " " + dot + " " + headerStyle.Render(fmt.Sprintf("%s %s (%s)", dayName, dateStr, relativeStr))
	}
	return headerStyle.Render(title) + " " + dot + " " + headerStyle.Render(fmt.Sprintf("%s %s", dayName, dateStr))
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
	count := filteredCommandCount(ctx.Commands, m.displayMode)
	countText := fmt.Sprintf("%d commands", count)
	if count == 1 {
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

func (m *Model) modeHint() string {
	switch m.displayMode {
	case UniqueMode:
		return "[u] *Uniq*  [m] Multi  [a] All"
	case MultiMode:
		return "[u] Uniq  [m] *Multi*  [a] All"
	default:
		return "[u] Uniq  [m] Multi"
	}
}

func (m *Model) renderStatusBar() string {
	hint := m.modeHint()
	if m.viewState == ContextDetailView {
		if len(m.detailCommands) == 0 {
			return statusBarStyle.Render("[Esc] Back  [h/l] Time  [H/L] Context  " + hint)
		}
		return statusBarStyle.Render("[j/k] Select  [Esc] Back  [h/l] Time  [H/L] Context  " + hint)
	}
	return statusBarStyle.Render("[j/k] Select  [Enter] View commands  [h/l] Time  [t] Today  [y] Yesterday  [q] Quit  " + hint)
}
