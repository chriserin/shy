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
	selectedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	normalStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	countStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	separatorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bucketLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)

	// Header/footer bar styles (ANSI 0-15 only, adapts to terminal colorscheme)
	barStyle       = lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("7"))
	barBoldStyle   = lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("15")).Bold(true)
	barAccentStyle = lipgloss.NewStyle().Background(lipgloss.Color("4")).Foreground(lipgloss.Color("0")).Bold(true)
	barDimStyle    = lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("4"))
	barBranchStyle = lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("14"))

	// Hint key style (no background, for empty-state navigation hints)
	hintKeyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)

	// Command detail styles (matching tv preview palette)
	detailLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // bright-blue
	detailErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // bright-red
	detailGitStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("13")) // bright-magenta

	// title styles
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	dayStyle   = lipgloss.NewStyle().Background(lipgloss.Color("0")).Foreground(lipgloss.Color("14"))
)

const marginX = 2

func (m *Model) renderView() string {
	switch m.viewState {
	case HelpView:
		return m.renderHelpView()
	case CommandDetailView:
		return m.renderCommandDetailView()
	case ContextDetailView:
		return m.renderDetailView()
	default:
		return m.renderSummaryView()
	}
}

func (m *Model) renderHelpView() string {
	var b strings.Builder

	margin := strings.Repeat(" ", marginX)

	// Render header bar for the previous view (so user retains context)
	b.WriteString(m.renderHelpHeaderBar())
	b.WriteString("\n")

	// Select bindings for the source view
	bindings := bindingsForView(m.helpPreviousView)

	// Find max key width for alignment
	maxKeyWidth := 0
	for _, bind := range bindings {
		if len(bind.key) > maxKeyWidth {
			maxKeyWidth = len(bind.key)
		}
	}

	b.WriteString("\n")
	b.WriteString(margin + titleStyle.Render("  Help") + "\n")
	b.WriteString("\n")

	contentLines := 3 // blank + title + blank
	for _, bind := range bindings {
		pad := maxKeyWidth - len(bind.key) + 2
		line := margin + "  " + hintKeyStyle.Render(bind.key) + strings.Repeat(" ", pad) + normalStyle.Render(bind.desc)
		b.WriteString(line + "\n")
		contentLines++
	}

	// Pad to push footer to bottom
	if m.height > 0 {
		avail := m.height - 2 // headerBar(1) + footerBar(1)
		if avail > contentLines {
			for i := 0; i < avail-contentLines; i++ {
				b.WriteString("\n")
			}
		}
	}

	// Footer bar
	b.WriteString(m.renderHelpFooterBar())

	return b.String()
}

func (m *Model) renderHelpHeaderBar() string {
	// Reuse the same header content as the source view
	// Temporarily swap viewState to render the correct header
	saved := m.viewState
	m.viewState = m.helpPreviousView
	header := m.renderHeaderBar()
	m.viewState = saved
	return header
}

func (m *Model) renderHelpFooterBar() string {
	content := barStyle.Render(" Press ") + barBoldStyle.Render("?") + barStyle.Render(" or ") + barBoldStyle.Render("esc") + barStyle.Render(" to close")
	contentWidth := ansi.StringWidth(content)
	pad := max(m.width-contentWidth, 0)
	return content + barStyle.Render(strings.Repeat(" ", pad))
}

func (m *Model) renderSummaryView() string {
	var b strings.Builder

	// Content width excludes left and right margins
	contentWidth := max(m.width-2*marginX, 20)
	margin := strings.Repeat(" ", marginX)

	// Header bar
	b.WriteString(m.renderHeaderBar())
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
			c := filteredCommandCount(ctx.Commands, m.displayMode, m.filterText)
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
	// Fixed lines: headerBar(1) + blank(1) + content + footerBar(1) = 3 + content
	if m.height > 0 {
		avail := m.height - 3
		if avail > contentLines {
			for i := 0; i < avail-contentLines; i++ {
				b.WriteString("\n")
			}
		}
	}

	// Footer bar
	b.WriteString(m.renderFooterBar())

	return b.String()
}

func (m *Model) renderDetailView() string {
	var b strings.Builder

	contentWidth := max(m.width-2*marginX, 20)
	margin := strings.Repeat(" ", marginX)

	// Header bar
	b.WriteString(m.renderHeaderBar())
	b.WriteString("\n")

	contentLines := 0
	if len(m.detailCommands) == 0 {
		emptyLines := m.renderEmptyDetailState(margin)
		for _, line := range emptyLines {
			b.WriteString(line + "\n")
		}
		contentLines = len(emptyLines)
	} else {
		// Build all body lines
		var bodyLines []string
		cmdIdx := 0
		for _, bucket := range m.detailBuckets {
			// Blank line before bucket
			bodyLines = append(bodyLines, "")
			// Bucket header
			label := bucketLabelStyle.Render(bucket.Label)
			dashWidth := max(contentWidth-2-ansi.StringWidth(bucket.Label)-1, 2)
			bodyLines = append(bodyLines, margin+"  "+label+" "+separatorStyle.Render(strings.Repeat("─", dashWidth)))
			// Commands
			for _, cmd := range bucket.Commands {
				bodyLines = append(bodyLines, margin+m.renderDetailCommand(cmd, cmdIdx == m.detailCmdIdx))
				cmdIdx++
			}
		}

		// Apply viewport scrolling if height is set
		if m.height > 0 {
			// headerBar(1) + footerBar(1)
			avail := max(m.height-2, 1)
			start := min(m.detailScrollOffset, len(bodyLines))
			end := min(start+avail, len(bodyLines))
			bodyLines = bodyLines[start:end]
		}

		for _, line := range bodyLines {
			b.WriteString(line)
			b.WriteString("\n")
		}
		contentLines = len(bodyLines)
	}

	// Pad to push footer to bottom
	// Fixed lines: headerBar(1) + content + footerBar(1) = 2 + content
	if m.height > 0 {
		avail := m.height - 2
		if avail > contentLines {
			for i := 0; i < avail-contentLines; i++ {
				b.WriteString("\n")
			}
		}
	}

	// Footer bar
	b.WriteString(m.renderFooterBar())

	return b.String()
}

// styledSegment is a piece of text with its visual width, used for word wrapping.
type styledSegment struct {
	text  string // rendered (may contain ANSI codes)
	width int    // visual width
}

// emptyDetailMessageLines builds styled "No commands found in ..." lines,
// wrapping at contentWidth. Data values are bold.
func (m *Model) emptyDetailMessageLines(contentWidth int) []string {
	bold := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	dim := normalStyle // white (not bold) for prose

	date := periodDateLabel(m.currentDate, m.period, m.now)

	segments := []styledSegment{
		{dim.Render("No commands found in "), ansi.StringWidth("No commands found in ")},
	}
	segments = append(segments, styledContextNameSegments(m.detailContextKey, m.detailContextBranch)...)
	segments = append(segments,
		styledSegment{dim.Render(" on "), ansi.StringWidth(" on ")},
		styledSegment{bold.Render(date), ansi.StringWidth(date)},
	)
	if m.filterText != "" {
		q := fmt.Sprintf("%q", m.filterText)
		segments = append(segments,
			styledSegment{dim.Render(" matching "), ansi.StringWidth(" matching ")},
			styledSegment{bold.Render(q), ansi.StringWidth(q)},
		)
	}
	if m.displayMode == UniqueMode {
		segments = append(segments,
			styledSegment{dim.Render(" (unique)"), ansi.StringWidth(" (unique)")},
		)
	}

	// Wrap segments into lines
	var lines []string
	var cur strings.Builder
	curWidth := 0
	for _, seg := range segments {
		if curWidth > 0 && curWidth+seg.width > contentWidth {
			lines = append(lines, cur.String())
			cur.Reset()
			curWidth = 0
		}
		cur.WriteString(seg.text)
		curWidth += seg.width
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// renderEmptyDetailState builds the lines for an empty detail view with
// navigation hints showing adjacent contexts (H/L) and periods (h/l).
func (m *Model) renderEmptyDetailState(margin string) []string {
	var lines []string

	contentWidth := m.width - 2*marginX

	lines = append(lines, "")
	for _, msgLine := range m.emptyDetailMessageLines(contentWidth) {
		lines = append(lines, margin+msgLine)
	}
	lines = append(lines, "")

	// h: previous period
	if m.emptyPrevPeriod != nil {
		lines = append(lines, margin+formatHintLine("h", m.emptyPrevPeriod.dateLabel, m.emptyPrevPeriod.count, contentWidth))
	} else {
		lines = append(lines, margin+formatHintLineDisabled("h"))
	}

	// l: next period
	if m.emptyNextPeriod != nil {
		lines = append(lines, margin+formatHintLine("l", m.emptyNextPeriod.dateLabel, m.emptyNextPeriod.count, contentWidth))
	} else {
		lines = append(lines, margin+formatHintLineDisabled("l"))
	}

	// H/L: context hints — in orphaned state these target last/first context
	orphaned := m.detailContextOrphaned()

	// H: previous context (or last context when orphaned)
	lines = append(lines, margin+m.contextHintLine("H", orphaned, len(m.contexts)-1, m.selectedIdx-1, contentWidth))

	// L: next context (or first context when orphaned)
	lines = append(lines, margin+m.contextHintLine("L", orphaned, 0, m.selectedIdx+1, contentWidth))

	return lines
}

// contextHintLine returns a formatted hint for a context navigation key (H/L).
// orphanedIdx is the context index to use when orphaned; neighborIdx is the
// adjacent context index to use otherwise.
func (m *Model) contextHintLine(key string, orphaned bool, orphanedIdx, neighborIdx int, width int) string {
	var ctx *ContextItem
	if orphaned && len(m.contexts) > 0 {
		ctx = &m.contexts[orphanedIdx]
	} else if !orphaned && neighborIdx >= 0 && neighborIdx < len(m.contexts) {
		ctx = &m.contexts[neighborIdx]
	}
	if ctx == nil {
		return formatHintLineDisabled(key)
	}
	name := formatContextName(ctx.Key, ctx.Branch)
	count := filteredCommandCount(ctx.Commands, m.displayMode, m.filterText)
	return formatHintLine(key, name, count, width)
}

// formatHintLine renders a navigation hint: key in accent (no bg), label in normal, count in dim.
func formatHintLine(key, label string, count int, width int) string {
	countText := fmt.Sprintf("%d commands", count)
	if count == 1 {
		countText = "1 command"
	}

	keyPart := hintKeyStyle.Render(" " + key + " ")
	labelPart := "  " + label

	// Compute padding to right-align count
	keyWidth := ansi.StringWidth(keyPart)
	labelWidth := ansi.StringWidth(labelPart)
	countWidth := len(countText)
	padding := max(width-keyWidth-labelWidth-countWidth, 1)

	return keyPart + normalStyle.Render(labelPart) + strings.Repeat(" ", padding) + countStyle.Render(countText)
}

// formatHintLineDisabled renders a dim hint line when the navigation option is unavailable.
func formatHintLineDisabled(key string) string {
	return countStyle.Render(" " + key + " ")
}

func (m *Model) renderDetailCommand(cmd models.Command, selected bool) string {
	t := time.Unix(cmd.Timestamp, 0)
	var minute string
	switch m.period {
	case MonthPeriod:
		minute = fmt.Sprintf("%s %2d:%s", t.Format("Mon"), hour12(t), t.Format("04 PM"))
	case DayPeriod:
		minute = t.Format(":04")
	default:
		minute = fmt.Sprintf("%2d:%s", hour12(t), t.Format("04 PM"))
	}

	cmdText := singleLine(cmd.CommandText)

	timeStr := "  " + minute + "  "

	if selected {
		return selectedStyle.Render("▶ ") + countStyle.Render(timeStr) + selectedStyle.Render(cmdText)
	}
	return countStyle.Render("  "+timeStr) + normalStyle.Render(cmdText)
}

func (m *Model) renderHeaderBar() string {
	// Focus indicator
	var focusSegment string
	if m.focused {
		focusSegment = barDimStyle.Render(" ● ")
	} else {
		focusSegment = barDimStyle.Render(" ○ ")
	}

	// Context/event info
	var infoSegment string
	switch m.viewState {
	case ContextDetailView:
		infoSegment = renderBarContextName(m.detailContextKey, m.detailContextBranch)
	case CommandDetailView:
		if target := m.CmdDetailTarget(); target != nil {
			infoSegment = barBoldStyle.Render(fmt.Sprintf(" Event: %d", target.ID))
		}
	}

	// Right side: date display + period indicator
	dateSegment := m.relativeDateIndicator() + barStyle.Render(" "+m.dateDisplayString())
	periodSegment := barAccentStyle.Render(" " + m.periodName() + " ")

	// Compose with padding
	left := focusSegment + infoSegment
	right := dateSegment + periodSegment

	leftWidth := ansi.StringWidth(left)
	rightWidth := ansi.StringWidth(right)
	padding := max(m.width-leftWidth-rightWidth, 0)

	return left + barStyle.Render(strings.Repeat(" ", padding)) + right
}

func (m *Model) dateDisplayString() string {
	currentYear := m.now().Year()

	switch m.period {
	case WeekPeriod:
		monday := mondayOfWeek(m.currentDate)
		return fmt.Sprintf("Week of %s ", formatShortDate(monday, currentYear))
	case MonthPeriod:
		return m.currentDate.Format("January 2006 ")
	default:
		dateStr := formatShortDate(m.currentDate, currentYear)
		dayName := m.currentDate.Format("Monday")
		return fmt.Sprintf("%s %s ", dayName, dateStr)
	}
}

// formatShortDate formats a date as "Jan 2" for the current year or "Jan 2, 2006" for past years.
func formatShortDate(t time.Time, currentYear int) string {
	if t.Year() == currentYear {
		return t.Format("Jan 2")
	}
	return t.Format("Jan 2, 2006")
}

// relativeDateIndicator returns a unicode marker for today/yesterday, empty otherwise.
func (m *Model) relativeDateIndicator() string {
	if m.period != DayPeriod {
		return ""
	}

	now := m.now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	current := time.Date(m.currentDate.Year(), m.currentDate.Month(), m.currentDate.Day(), 0, 0, 0, 0, time.Local)

	diff := today.Sub(current)
	days := int(diff.Hours() / 24)

	switch days {
	case 0:
		return dayStyle.Render("TODAY")
	case 1:
		return dayStyle.Render("YESTERDAY")
	default:
		return ""
	}
}

func (m *Model) renderFooterBar() string {
	if m.filterActive {
		content := barStyle.Render(fmt.Sprintf(" Filter: %s█", m.filterText))
		contentWidth := ansi.StringWidth(content)
		pad := max(m.width-contentWidth, 0)
		return content + barStyle.Render(strings.Repeat(" ", pad))
	}

	// Left: mode indicator (not shown in command detail view) + filter indicator
	var left string
	if m.viewState != CommandDetailView {
		left = barAccentStyle.Render(" " + m.activeModeName() + " ")
	}
	if m.filterText != "" {
		left += barStyle.Render(" /" + m.filterText + " ")
	}

	// Right: status flash message
	var right string
	if m.statusMsg != "" {
		right = barDimStyle.Render(" " + m.statusMsg + " ")
	}

	leftWidth := ansi.StringWidth(left)
	rightWidth := ansi.StringWidth(right)
	padding := max(m.width-leftWidth-rightWidth, 0)

	return left + barStyle.Render(strings.Repeat(" ", padding)) + right
}

func (m *Model) activeModeName() string {
	switch m.displayMode {
	case UniqueMode:
		return "Uniq"
	default:
		return "All"
	}
}

func (m *Model) periodName() string {
	switch m.period {
	case WeekPeriod:
		return "Week"
	case MonthPeriod:
		return "Month"
	default:
		return "Day"
	}
}

func (m *Model) renderContextItem(ctx ContextItem, selected bool, width int, countWidth int) string {
	// Format command count
	count := filteredCommandCount(ctx.Commands, m.displayMode, m.filterText)
	countText := fmt.Sprintf("%d commands", count)
	if count == 1 {
		countText = "1 command "
	}

	prefix := "  "
	if selected {
		prefix = "▶ "
	}

	// Available space for name: width - prefix(2) - gap(2) - countWidth
	gap := 2
	nameMaxWidth := max(width-len(prefix)-gap-countWidth, 10)

	// Build styled context name with green branch
	name := styledSummaryContextName(ctx.Key, ctx.Branch, selected)
	name = truncateWithEllipsis(name, nameMaxWidth)

	// Build the line with right-aligned count
	padding := max(width-ansi.StringWidth(prefix)-ansi.StringWidth(name)-ansi.StringWidth(countText), 1)

	if selected {
		return selectedStyle.Render(prefix) + name + strings.Repeat(" ", padding) + selectedStyle.Render(countText)
	}
	return normalStyle.Render(prefix) + name + strings.Repeat(" ", padding) + countStyle.Render(countText)
}

// styledSummaryContextName renders a context name with the branch in cyan.
func styledSummaryContextName(key summary.ContextKey, branch summary.BranchKey, selected bool) string {
	dir := formatDir(key.WorkingDir)
	branchCyanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	dirStyle, sepStyle, brStyle := normalStyle, countStyle, branchCyanStyle
	if selected {
		dirStyle = selectedStyle
	}
	if hasBranch(key, branch) {
		return dirStyle.Render(dir) + sepStyle.Render(":") + brStyle.Render(string(branch))
	}
	return dirStyle.Render(dir)
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

func (m *Model) renderCommandDetailView() string {
	var b strings.Builder

	contentWidth := max(m.width-2*marginX, 20)
	margin := strings.Repeat(" ", marginX)

	// Header bar
	b.WriteString(m.renderHeaderBar())
	b.WriteString("\n")

	target := m.CmdDetailTarget()
	if target == nil {
		b.WriteString("\n" + margin + "No command selected\n")
	} else {
		cmd := target

		b.WriteString("\n")
		// Metadata fields (label in blue, value in white — matching tv preview)
		b.WriteString(margin + "  " + renderDetailField("Command:", singleLine(cmd.CommandText), normalStyle) + "\n")
		b.WriteString(margin + "  " + renderDetailField("Working Dir:", formatDir(cmd.WorkingDir), normalStyle) + "\n")

		if cmd.GitRepo != nil {
			b.WriteString(margin + "  " + renderDetailField("Git Repo:", *cmd.GitRepo, detailGitStyle) + "\n")
		} else {
			b.WriteString(margin + "  " + renderDetailField("Git Repo:", "-", normalStyle) + "\n")
		}
		if cmd.GitBranch != nil {
			b.WriteString(margin + "  " + renderDetailField("Git Branch:", *cmd.GitBranch, detailGitStyle) + "\n")
		} else {
			b.WriteString(margin + "  " + renderDetailField("Git Branch:", "-", normalStyle) + "\n")
		}

		if cmd.SourceApp != nil && cmd.SourcePid != nil {
			sessionStr := fmt.Sprintf("%s:%d", *cmd.SourceApp, *cmd.SourcePid)
			b.WriteString(margin + "  " + renderDetailField("Session:", sessionStr, normalStyle) + "\n")
		} else {
			b.WriteString(margin + "  " + renderDetailField("Session:", "-", normalStyle) + "\n")
		}

		t := time.Unix(cmd.Timestamp, 0)
		b.WriteString(margin + "  " + renderDetailField("Timestamp:", t.Format("2006-01-02 15:04"), normalStyle) + "\n")
		b.WriteString(margin + "  " + renderDetailField("Duration:", formatDurationHuman(cmd.Duration), normalStyle) + "\n")
		b.WriteString(margin + "  " + renderDetailField("Exit Status:", renderExitStatus(cmd.ExitStatus), lipgloss.NewStyle()) + "\n")

		// Separator
		b.WriteString("\n")
		b.WriteString(margin + "  " + separatorStyle.Render(strings.Repeat("─", contentWidth-4)) + "\n")
		b.WriteString("\n")

		// Session context
		b.WriteString(margin + "  " + detailLabelStyle.Render("Context (same session):") + "\n")

		allCmds := m.cmdDetailAllCommands()
		for i, ctxCmd := range allCmds {
			cmdText := singleLine(ctxCmd.CommandText)
			idStr := fmt.Sprintf("%5d  ", ctxCmd.ID)
			if i == m.cmdDetailIdx {
				b.WriteString(margin + "  " + selectedStyle.Render("▶ ") + countStyle.Render(idStr) + selectedStyle.Render(cmdText) + "\n")
			} else {
				b.WriteString(margin + "  " + countStyle.Render("  "+idStr) + normalStyle.Render(cmdText) + "\n")
			}
		}
	}

	// Pad to push footer to bottom
	// Fixed lines: headerBar(1) + content + footerBar(1) = 2 + content
	contentLines := 0
	if target != nil {
		// blank + 5 metadata + 2 git + 1 session + blank + separator + blank + "Context" + context cmds
		contentLines = 1 + 5 + 2 + 1
		contentLines += 3 + 1 // blank + separator + blank + "Context"
		contentLines += len(m.cmdDetailAllCommands())
	} else {
		contentLines = 2
	}

	if m.height > 0 {
		avail := m.height - 2
		if avail > contentLines {
			for i := 0; i < avail-contentLines; i++ {
				b.WriteString("\n")
			}
		}
	}

	// Footer bar
	b.WriteString(m.renderFooterBar())

	return b.String()
}

// renderDetailField renders a label:value pair with the label in blue and the
// value in the given style, padded to align at column 13.
func renderDetailField(label string, value string, valueStyle lipgloss.Style) string {
	const padTo = 13
	padding := max(padTo-len(label), 1)
	return detailLabelStyle.Render(label) + strings.Repeat(" ", padding) + valueStyle.Render(value)
}

// hour12 returns the 12-hour clock value (1-12) for the given time.
func hour12(t time.Time) int {
	h := t.Hour() % 12
	if h == 0 {
		h = 12
	}
	return h
}

// singleLine collapses a multi-line command to its first line with a ↵ indicator.
func singleLine(s string) string {
	if parts := strings.SplitN(s, "\n", 2); len(parts) > 1 {
		return parts[0] + " ↵"
	}
	return s
}

// renderExitStatus returns a colored exit status string (green for 0, red for non-zero).
func renderExitStatus(code int) string {
	var unicodeCheckmark = "\u2713"
	var unicodeX = "\u2717"
	if code == 0 {
		return selectedStyle.Render("0 " + unicodeCheckmark)
	}
	return detailErrorStyle.Render(fmt.Sprintf("%d %s", code, unicodeX))
}

// formatDurationHuman formats a duration in milliseconds to human-readable form
func formatDurationHuman(durationMs *int64) string {
	var emDash = "\u2014"
	if durationMs == nil {
		return emDash
	}

	millis := *durationMs
	d := time.Duration(millis) * time.Millisecond

	if d < time.Second {
		return fmt.Sprintf("%dms", millis)
	}

	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	d -= minutes * time.Minute
	seconds := d / time.Second

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// hasBranch returns true when the context has a git repo with a real branch.
func hasBranch(key summary.ContextKey, branch summary.BranchKey) bool {
	return key.GitRepo != "" && branch != summary.NoBranch
}

// styledContextNameSegments returns styled segments for a context name with
// distinct styles for directory, separator, and branch (no background).
func styledContextNameSegments(key summary.ContextKey, branch summary.BranchKey) []styledSegment {
	bold := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	branchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	dir := formatDir(key.WorkingDir)
	if hasBranch(key, branch) {
		b := string(branch)
		return []styledSegment{
			{bold.Render(dir), ansi.StringWidth(dir)},
			{sep.Render(":"), 1},
			{branchStyle.Render(b), ansi.StringWidth(b)},
		}
	}
	return []styledSegment{
		{bold.Render(dir), ansi.StringWidth(dir)},
	}
}

// renderBarContextName renders the context name for the header bar with
// distinct styles for directory, separator, and branch.
func renderBarContextName(key summary.ContextKey, branch summary.BranchKey) string {
	dir := formatDir(key.WorkingDir)
	if hasBranch(key, branch) {
		return barBoldStyle.Render(" "+dir) +
			barDimStyle.Render(":") +
			barBranchStyle.Render(string(branch))
	}
	return barBoldStyle.Render(" " + dir)
}

func formatContextName(key summary.ContextKey, branch summary.BranchKey) string {
	dir := formatDir(key.WorkingDir)
	if hasBranch(key, branch) {
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
