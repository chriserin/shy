package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/internal/summary"
	"github.com/chris/shy/pkg/models"
)

// ViewState represents which view is currently displayed
type ViewState int

const (
	SummaryView ViewState = iota
	ContextDetailView
	CommandDetailView
	HelpView
)

// DisplayMode controls which commands are shown based on frequency
type DisplayMode int

const (
	AllMode    DisplayMode = iota
	UniqueMode
)

// Period represents the time granularity for the view
type Period int

const (
	DayPeriod   Period = iota
	WeekPeriod
	MonthPeriod
)

// periodPeekData holds a label and command count for an adjacent period
type periodPeekData struct {
	dateLabel string
	count     int
}

func (p *periodPeekData) DateLabel() string { return p.dateLabel }
func (p *periodPeekData) Count() int        { return p.count }

// DetailBucket represents a time bucket with its label and commands
type DetailBucket struct {
	Label    string
	Commands []models.Command
}

// ContextItem represents a context with its command count
type ContextItem struct {
	Key          summary.ContextKey
	Branch       summary.BranchKey
	CommandCount int
	Commands     []models.Command
}

// Model represents the TUI state
type Model struct {
	// Database
	db     *db.DB
	dbPath string

	// Data
	contexts    []ContextItem
	currentDate time.Time

	// View state
	viewState        ViewState
	helpPreviousView ViewState
	detailBuckets        []DetailBucket
	detailCommands       []models.Command
	detailCmdIdx         int
	detailScrollOffset   int
	detailContextKey     summary.ContextKey
	detailContextBranch  summary.BranchKey
	pendingDetailReentry bool

	// Empty state peek data (adjacent period hints)
	emptyPrevPeriod *periodPeekData
	emptyNextPeriod *periodPeekData

	// Command detail view
	cmdDetailAll      []models.Command // full session context: [before..., target, after...]
	cmdDetailIdx      int              // index of currently selected command
	cmdDetailStartIdx int              // index of the original target in cmdDetailAll

	// Period
	period     Period    // current period (default DayPeriod)
	anchorDate time.Time // saved day-level date for Week→Day restore

	// Filter
	filterText     string // currently active filter (persists across views)
	filterActive   bool   // whether the filter input bar is open
	filterPrevText string // saved before opening bar, for Esc cancel

	// Display mode
	displayMode DisplayMode

	// Selection
	selectedIdx int

	// UI dimensions
	width  int
	height int

	// Focus
	focused bool

	// For testing - allows injecting "today"
	now func() time.Time
}

// Option is a functional option for configuring the Model
type Option func(*Model)

// WithNow sets the function used to get the current time (for testing)
func WithNow(fn func() time.Time) Option {
	return func(m *Model) {
		m.now = fn
	}
}

// New creates a new Model
func New(dbPath string, opts ...Option) *Model {
	m := &Model{
		dbPath:      dbPath,
		currentDate: time.Now().AddDate(0, 0, -1), // Yesterday
		selectedIdx: 0,
		focused:     true,
		now:         time.Now,
	}

	for _, opt := range opts {
		opt(m)
	}

	// Recalculate yesterday based on now function
	m.currentDate = m.now().AddDate(0, 0, -1)

	return m
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return m.loadContexts
}

// loadContexts loads contexts for the current date
func (m *Model) loadContexts() tea.Msg {
	database, err := db.New(m.dbPath)
	if err != nil {
		return errMsg{err}
	}
	defer database.Close()

	startTime, endTime := m.dateRange()
	commands, err := database.GetCommandsByDateRange(startTime, endTime, nil)
	if err != nil {
		return errMsg{err}
	}

	// Group by context
	grouped := summary.GroupByContext(commands)

	// Convert to ContextItems
	var items []ContextItem
	for ctxKey, branches := range grouped.Contexts {
		for branchKey, cmds := range branches {
			items = append(items, ContextItem{
				Key:          ctxKey,
				Branch:       branchKey,
				CommandCount: len(cmds),
				Commands:     cmds,
			})
		}
	}

	// Sort contexts alphabetically by working dir, then branch
	sortContextItems(items)

	return contextsLoadedMsg{contexts: items}
}

// dateRangeForPeriod returns the start and end timestamps for the given date and period.
func dateRangeForPeriod(date time.Time, period Period) (int64, int64) {
	year, month, day := date.Date()

	switch period {
	case WeekPeriod:
		// Monday 00:00 → next Monday 00:00
		d := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
		weekday := d.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		monday := d.AddDate(0, 0, -int(weekday-time.Monday))
		return monday.Unix(), monday.AddDate(0, 0, 7).Unix()

	case MonthPeriod:
		// 1st of month 00:00 → 1st of next month 00:00
		startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
		endOfMonth := startOfMonth.AddDate(0, 1, 0)
		return startOfMonth.Unix(), endOfMonth.Unix()

	default: // DayPeriod
		startOfDay := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
		endOfDay := startOfDay.AddDate(0, 0, 1)
		return startOfDay.Unix(), endOfDay.Unix()
	}
}

// dateRange returns the start and end timestamps for the current period
func (m *Model) dateRange() (int64, int64) {
	return dateRangeForPeriod(m.currentDate, m.period)
}

// adjacentDate returns the date shifted by one period in the given direction (-1 or +1).
func adjacentDate(date time.Time, period Period, direction int) time.Time {
	switch period {
	case WeekPeriod:
		return date.AddDate(0, 0, 7*direction)
	case MonthPeriod:
		return date.AddDate(0, direction, 0)
	default:
		return date.AddDate(0, 0, direction)
	}
}

// isCurrentPeriod returns true if currentDate falls within the current period of "now"
func (m *Model) isCurrentPeriod() bool {
	now := m.now()
	switch m.period {
	case WeekPeriod:
		nowY, nowW := now.ISOWeek()
		curY, curW := m.currentDate.ISOWeek()
		return nowY == curY && nowW == curW
	case MonthPeriod:
		return now.Year() == m.currentDate.Year() && now.Month() == m.currentDate.Month()
	default: // DayPeriod
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		currentStart := time.Date(m.currentDate.Year(), m.currentDate.Month(), m.currentDate.Day(), 0, 0, 0, 0, time.Local)
		return !currentStart.Before(todayStart)
	}
}

// navigateBack moves the date backward by one period unit
func (m *Model) navigateBack() {
	switch m.period {
	case WeekPeriod:
		m.currentDate = m.currentDate.AddDate(0, 0, -7)
	case MonthPeriod:
		m.currentDate = m.currentDate.AddDate(0, -1, 0)
	default:
		m.currentDate = m.currentDate.AddDate(0, 0, -1)
	}
}

// navigateForward moves the date forward by one period unit
func (m *Model) navigateForward() {
	switch m.period {
	case WeekPeriod:
		m.currentDate = m.currentDate.AddDate(0, 0, 7)
	case MonthPeriod:
		m.currentDate = m.currentDate.AddDate(0, 1, 0)
	default:
		m.currentDate = m.currentDate.AddDate(0, 0, 1)
	}
}

// cyclePeriodUp moves Day→Week→Month
func (m *Model) cyclePeriodUp() bool {
	switch m.period {
	case DayPeriod:
		m.anchorDate = m.currentDate
		m.period = WeekPeriod
		return true
	case WeekPeriod:
		m.period = MonthPeriod
		return true
	default:
		return false
	}
}

// cyclePeriodDown moves Month→Week→Day
func (m *Model) cyclePeriodDown() bool {
	switch m.period {
	case MonthPeriod:
		m.period = WeekPeriod
		return true
	case WeekPeriod:
		m.period = DayPeriod
		if !m.anchorDate.IsZero() {
			m.currentDate = m.anchorDate
		}
		return true
	default:
		return false
	}
}

// cmdDetailTotalContext returns the total number of context lines (before + after)
// available for the command detail view.
func (m *Model) cmdDetailTotalContext() int {
	if m.height <= 0 {
		return 10 // fallback
	}
	// 16 = frame(2) + blank(1) + metadata(8) + blank-sep-blank-label(4) + target command(1)
	avail := m.height - 16
	if avail < 1 {
		return 1
	}
	return avail
}

// balanceContext trims before/after slices so their combined length fits within
// total, while maximizing the number of commands shown. When one side is short,
// the surplus goes to the other.
func balanceContext(before, after []models.Command, total int) ([]models.Command, []models.Command) {
	if len(before)+len(after) <= total {
		return before, after
	}
	half := total / 2
	bLen, aLen := len(before), len(after)
	if bLen <= half {
		// before is short — give surplus to after
		aMax := total - bLen
		if aLen > aMax {
			after = after[:aMax]
		}
	} else if aLen <= half {
		// after is short — give surplus to before
		bMax := total - aLen
		if bLen > bMax {
			before = before[bLen-bMax:]
		}
	} else {
		// both sides overflow — split evenly
		before = before[bLen-half:]
		after = after[:total-half]
	}
	return before, after
}

// loadCommandContext loads session context for a specific command
func (m *Model) loadCommandContext(cmdID int64) tea.Cmd {
	return func() tea.Msg {
		database, err := db.New(m.dbPath)
		if err != nil {
			return errMsg{err}
		}
		defer database.Close()

		total := m.cmdDetailTotalContext()
		before, target, after, err := database.GetCommandWithContext(cmdID, total)
		if err != nil {
			return errMsg{err}
		}

		before, after = balanceContext(before, after, total)

		return commandContextLoadedMsg{
			before: before,
			target: target,
			after:  after,
		}
	}
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.viewState == CommandDetailView && m.cmdDetailIdx < len(m.cmdDetailAll) {
			return m, m.loadCommandContext(m.cmdDetailAll[m.cmdDetailIdx].ID)
		}
		return m, nil

	case tea.FocusMsg:
		m.focused = true
		return m, nil

	case tea.BlurMsg:
		m.focused = false
		return m, nil

	case contextsLoadedMsg:
		m.contexts = msg.contexts
		m.selectedIdx = 0
		if m.pendingDetailReentry {
			m.pendingDetailReentry = false
			found := false
			for i, ctx := range m.contexts {
				if ctx.Key == m.detailContextKey && ctx.Branch == m.detailContextBranch {
					m.selectedIdx = i
					return m, m.enterDetailView()
				}
			}
			if !found {
				// Context not on this day — show empty detail view
				m.viewState = ContextDetailView
				m.detailBuckets = nil
				m.detailCommands = nil
				m.detailCmdIdx = 0
				m.detailScrollOffset = 0
				m.emptyPrevPeriod = nil
				m.emptyNextPeriod = nil
				return m, m.loadEmptyStatePeeks()
			}
		}
		return m, nil

	case commandContextLoadedMsg:
		var all []models.Command
		all = append(all, msg.before...)
		if msg.target != nil {
			all = append(all, *msg.target)
		}
		all = append(all, msg.after...)
		m.cmdDetailAll = all
		m.cmdDetailIdx = len(msg.before) // point at target
		if m.viewState != CommandDetailView {
			m.cmdDetailStartIdx = m.cmdDetailIdx
		}
		m.viewState = CommandDetailView
		return m, nil

	case emptyStatePeeksMsg:
		m.emptyPrevPeriod = msg.prev
		m.emptyNextPeriod = msg.next
		return m, nil

	case errMsg:
		// TODO: handle error display
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	if m.filterActive {
		return m.handleFilterKey(msg)
	}

	// ESC clears filter when one is active (in any view)
	if msg.String() == "esc" && m.filterText != "" {
		m.filterText = ""
		if m.viewState == ContextDetailView {
			return m, m.refreshDetailView()
		}
		return m, nil
	}

	switch m.viewState {
	case HelpView:
		return m.handleHelpKey(msg)
	case CommandDetailView:
		return m.handleCommandDetailKey(msg)
	case ContextDetailView:
		return m.handleDetailKey(msg)
	default:
		return m.handleSummaryKey(msg)
	}
}

func (m *Model) handleSummaryKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.selectedIdx < len(m.contexts)-1 {
			m.selectedIdx++
		}
		return m, nil

	case "k", "up":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return m, nil

	case "enter":
		if len(m.contexts) > 0 {
			return m, m.enterDetailView()
		}
		return m, nil

	case "h":
		m.navigateBack()
		m.selectedIdx = 0
		return m, m.loadContexts

	case "l":
		if m.isCurrentPeriod() {
			return m, nil
		}
		m.navigateForward()
		m.selectedIdx = 0
		return m, m.loadContexts

	case "t":
		m.currentDate = m.now()
		m.period = DayPeriod
		m.selectedIdx = 0
		return m, m.loadContexts

	case "y":
		m.currentDate = m.now().AddDate(0, 0, -1)
		m.period = DayPeriod
		m.selectedIdx = 0
		return m, m.loadContexts

	case "u":
		m.displayMode = UniqueMode
		return m, nil

	case "a":
		m.displayMode = AllMode
		return m, nil

	case "/":
		m.filterActive = true
		m.filterPrevText = m.filterText
		return m, nil

	case "]":
		if m.cyclePeriodUp() {
			m.selectedIdx = 0
			return m, m.loadContexts
		}
		return m, nil

	case "[":
		if m.cyclePeriodDown() {
			m.selectedIdx = 0
			return m, m.loadContexts
		}
		return m, nil

	case "?":
		m.helpPreviousView = m.viewState
		m.viewState = HelpView
		return m, nil
	}

	return m, nil
}

func (m *Model) handleDetailKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.detailCmdIdx < len(m.detailCommands)-1 {
			m.detailCmdIdx++
			m.ensureDetailCmdVisible()
		}
		return m, nil

	case "k", "up":
		if m.detailCmdIdx > 0 {
			m.detailCmdIdx--
			m.ensureDetailCmdVisible()
		}
		return m, nil

	case "enter":
		if len(m.detailCommands) > 0 {
			cmd := m.detailCommands[m.detailCmdIdx]
			return m, m.loadCommandContext(cmd.ID)
		}
		return m, nil

	case "-":
		m.viewState = SummaryView
		return m, nil

	case "H":
		// Switch to previous context
		if m.detailContextOrphaned() {
			if len(m.contexts) > 0 {
				m.selectedIdx = len(m.contexts) - 1
				return m, m.enterDetailView()
			}
		} else if m.selectedIdx > 0 {
			m.selectedIdx--
			return m, m.enterDetailView()
		}
		return m, nil

	case "L":
		// Switch to next context
		if m.detailContextOrphaned() {
			if len(m.contexts) > 0 {
				m.selectedIdx = 0
				return m, m.enterDetailView()
			}
		} else if m.selectedIdx < len(m.contexts)-1 {
			m.selectedIdx++
			return m, m.enterDetailView()
		}
		return m, nil

	case "h":
		m.navigateBack()
		m.pendingDetailReentry = true
		return m, m.loadContexts

	case "l":
		if m.isCurrentPeriod() {
			return m, nil
		}
		m.navigateForward()
		m.pendingDetailReentry = true
		return m, m.loadContexts

	case "t":
		m.currentDate = m.now()
		m.period = DayPeriod
		m.selectedIdx = 0
		m.viewState = SummaryView
		return m, m.loadContexts

	case "y":
		m.currentDate = m.now().AddDate(0, 0, -1)
		m.period = DayPeriod
		m.selectedIdx = 0
		m.viewState = SummaryView
		return m, m.loadContexts

	case "u":
		m.displayMode = UniqueMode
		return m, m.refreshDetailView()

	case "a":
		m.displayMode = AllMode
		return m, m.refreshDetailView()

	case "/":
		m.filterActive = true
		m.filterPrevText = m.filterText
		return m, nil

	case "]":
		if m.cyclePeriodUp() {
			m.pendingDetailReentry = true
			return m, m.loadContexts
		}
		return m, nil

	case "[":
		if m.cyclePeriodDown() {
			m.pendingDetailReentry = true
			return m, m.loadContexts
		}
		return m, nil

	case "?":
		m.helpPreviousView = m.viewState
		m.viewState = HelpView
		return m, nil
	}

	return m, nil
}

// cmdDetailAllCommands returns the full session context list
func (m *Model) cmdDetailAllCommands() []models.Command {
	return m.cmdDetailAll
}

func (m *Model) handleCommandDetailKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.cmdDetailIdx < len(m.cmdDetailAll)-1 {
			nextCmd := m.cmdDetailAll[m.cmdDetailIdx+1]
			return m, m.loadCommandContext(nextCmd.ID)
		}
		return m, nil

	case "k", "up":
		if m.cmdDetailIdx > 0 {
			prevCmd := m.cmdDetailAll[m.cmdDetailIdx-1]
			return m, m.loadCommandContext(prevCmd.ID)
		}
		return m, nil

	case "-":
		// Return to ContextDetailView, restore selection to viewed command
		m.viewState = ContextDetailView
		if m.cmdDetailIdx < len(m.cmdDetailAll) {
			target := m.cmdDetailAll[m.cmdDetailIdx]
			for i, cmd := range m.detailCommands {
				if cmd.ID == target.ID {
					m.detailCmdIdx = i
					m.ensureDetailCmdVisible()
					break
				}
			}
		}
		return m, nil

	case "?":
		m.helpPreviousView = m.viewState
		m.viewState = HelpView
		return m, nil
	}

	return m, nil
}

func (m *Model) handleHelpKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?", "esc":
		m.viewState = m.helpPreviousView
		return m, nil
	}
	return m, nil
}

func (m *Model) handleFilterKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyEnter:
		m.filterActive = false
		// If in detail view, re-enter to apply filter
		if m.viewState == ContextDetailView {
			return m, m.refreshDetailView()
		}
		return m, nil

	case tea.KeyEscape:
		m.filterActive = false
		m.filterText = m.filterPrevText
		// If in detail view, re-enter to restore filter
		if m.viewState == ContextDetailView {
			return m, m.refreshDetailView()
		}
		return m, nil

	case tea.KeyBackspace:
		if len(m.filterText) == 0 {
			m.filterActive = false
			return m, nil
		}
		// Remove last rune
		runes := []rune(m.filterText)
		m.filterText = string(runes[:len(runes)-1])
		// Live update in detail view
		if m.viewState == ContextDetailView {
			return m, m.refreshDetailView()
		}
		return m, nil

	case tea.KeySpace:
		m.filterText += " "
		if m.viewState == ContextDetailView {
			return m, m.refreshDetailView()
		}
		return m, nil

	case tea.KeyRunes:
		m.filterText += string(msg.Runes)
		// Live update in detail view
		if m.viewState == ContextDetailView {
			return m, m.refreshDetailView()
		}
		return m, nil
	}

	return m, nil
}

// detailContextOrphaned returns true when the detail view's context is not
// present in the current contexts list (e.g. after navigating to a period
// where the context has no commands).
func (m *Model) detailContextOrphaned() bool {
	if m.selectedIdx >= len(m.contexts) {
		return true
	}
	ctx := m.contexts[m.selectedIdx]
	return ctx.Key != m.detailContextKey || ctx.Branch != m.detailContextBranch
}

// refreshDetailView re-applies filters to the current detail context without
// switching to a different context. Used by filter handlers and ESC.
func (m *Model) refreshDetailView() tea.Cmd {
	if m.detailContextOrphaned() {
		m.emptyPrevPeriod = nil
		m.emptyNextPeriod = nil
		m.detailBuckets = nil
		m.detailCommands = nil
		m.detailCmdIdx = 0
		m.detailScrollOffset = 0
		return m.loadEmptyStatePeeks()
	}
	return m.enterDetailView()
}

func (m *Model) enterDetailView() tea.Cmd {
	m.emptyPrevPeriod = nil
	m.emptyNextPeriod = nil

	if len(m.contexts) == 0 || m.selectedIdx >= len(m.contexts) {
		m.viewState = ContextDetailView
		m.detailBuckets = nil
		m.detailCommands = nil
		m.detailCmdIdx = 0
		m.detailScrollOffset = 0
		return m.loadEmptyStatePeeks()
	}

	ctx := m.contexts[m.selectedIdx]

	m.detailContextKey = ctx.Key
	m.detailContextBranch = ctx.Branch

	// Apply substring filter first, then mode filter
	subFiltered := filterBySubstring(ctx.Commands, m.filterText)
	filtered := filterByMode(subFiltered, m.displayMode)

	// Bucket size depends on period
	var bucketSize summary.BucketSize
	switch m.period {
	case WeekPeriod:
		bucketSize = summary.Daily
	case MonthPeriod:
		bucketSize = summary.Weekly
	default:
		bucketSize = summary.Hourly
	}

	bucketMap := summary.BucketBy(filtered, bucketSize)
	orderedIDs := summary.GetOrderedBuckets(bucketMap)

	// Build detail buckets and flat command list
	var buckets []DetailBucket
	var flatCommands []models.Command

	for _, id := range orderedIDs {
		bucket := bucketMap[id]

		// Format label based on period
		var label string
		switch m.period {
		case WeekPeriod:
			t := time.Unix(int64(id), 0).Local()
			label = t.Format("Mon Jan 2")
		case MonthPeriod:
			// Derive the Monday from the first command in this bucket
			if len(bucket.Commands) > 0 {
				t := time.Unix(bucket.Commands[0].Timestamp, 0).Local()
				weekday := t.Weekday()
				if weekday == time.Sunday {
					weekday = 7
				}
				monday := t.AddDate(0, 0, -int(weekday-time.Monday))
				label = fmt.Sprintf("Week of %s", monday.Format("Jan 2"))
			} else {
				label = fmt.Sprintf("Week %d", id)
			}
		default:
			label = summary.FormatHour(id)
		}

		// Sort commands within bucket by timestamp
		cmds := make([]models.Command, len(bucket.Commands))
		copy(cmds, bucket.Commands)
		sort.Slice(cmds, func(i, j int) bool {
			return cmds[i].Timestamp < cmds[j].Timestamp
		})

		buckets = append(buckets, DetailBucket{
			Label:    label,
			Commands: cmds,
		})
		flatCommands = append(flatCommands, cmds...)
	}

	m.viewState = ContextDetailView
	m.detailBuckets = buckets
	m.detailCommands = flatCommands
	m.detailCmdIdx = 0
	m.detailScrollOffset = 0

	if len(flatCommands) == 0 {
		return m.loadEmptyStatePeeks()
	}
	return nil
}

// loadEmptyStatePeeks returns an async command that queries adjacent periods
// for the current context and returns peek data (date label + command count).
func (m *Model) loadEmptyStatePeeks() tea.Cmd {
	dbPath := m.dbPath
	ctxKey := m.detailContextKey
	ctxBranch := m.detailContextBranch
	curDate := m.currentDate
	period := m.period
	mode := m.displayMode
	filter := m.filterText
	nowFn := m.now
	isCurrentPeriod := m.isCurrentPeriod()

	return func() tea.Msg {
		database, err := db.New(dbPath)
		if err != nil {
			return emptyStatePeeksMsg{}
		}
		defer database.Close()

		peekPeriod := func(date time.Time) *periodPeekData {
			start, end := dateRangeForPeriod(date, period)
			commands, err := database.GetCommandsByDateRange(start, end, nil)
			if err != nil {
				return nil
			}
			grouped := summary.GroupByContext(commands)
			if branches, ok := grouped.Contexts[ctxKey]; ok {
				if cmds, ok := branches[ctxBranch]; ok {
					count := filteredCommandCount(cmds, mode, filter)
					return &periodPeekData{
						dateLabel: periodDateLabel(date, period, nowFn),
						count:     count,
					}
				}
			}
			return &periodPeekData{
				dateLabel: periodDateLabel(date, period, nowFn),
				count:     0,
			}
		}

		prevDate := adjacentDate(curDate, period, -1)
		prev := peekPeriod(prevDate)

		var next *periodPeekData
		if !isCurrentPeriod {
			nextDate := adjacentDate(curDate, period, 1)
			next = peekPeriod(nextDate)
		}

		return emptyStatePeeksMsg{prev: prev, next: next}
	}
}

// periodDateLabel formats a date label for a period, similar to dateDisplayString
// but without trailing spaces or indicators.
func periodDateLabel(date time.Time, period Period, nowFn func() time.Time) string {
	currentYear := nowFn().Year()

	switch period {
	case WeekPeriod:
		year, month, day := date.Date()
		d := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
		weekday := d.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		monday := d.AddDate(0, 0, -int(weekday-time.Monday))
		if monday.Year() == currentYear {
			return fmt.Sprintf("Week of %s", monday.Format("Jan 2"))
		}
		return fmt.Sprintf("Week of %s", monday.Format("Jan 2, 2006"))
	case MonthPeriod:
		return date.Format("January 2006")
	default:
		dayName := date.Format("Mon")
		if date.Year() == currentYear {
			return fmt.Sprintf("%s %s", dayName, date.Format("Jan 2"))
		}
		return fmt.Sprintf("%s %s", dayName, date.Format("Jan 2, 2006"))
	}
}

// detailCmdBodyLine returns the body-line index and bucket-start line of the
// currently selected command. bucketStart points to the blank line before the
// bucket header, so scrolling to it reveals the full bucket context.
func (m *Model) detailCmdBodyLine() (cmdLine int, bucketStart int) {
	line := 0
	cmdSeen := 0
	for _, bucket := range m.detailBuckets {
		bStart := line
		line++ // blank before bucket
		line++ // bucket header
		for range bucket.Commands {
			if cmdSeen == m.detailCmdIdx {
				return line, bStart
			}
			line++
			cmdSeen++
		}
	}
	return line, 0
}

// ensureDetailCmdVisible adjusts detailScrollOffset to keep selected command in view
func (m *Model) ensureDetailCmdVisible() {
	if m.height == 0 || len(m.detailCommands) == 0 {
		return
	}
	avail := m.height - 2 // headerBar(1) + footerBar(1)
	if avail < 1 {
		avail = 1
	}
	cmdLine, bucketStart := m.detailCmdBodyLine()
	if cmdLine < m.detailScrollOffset {
		// Scroll up: show the bucket header context above the command
		m.detailScrollOffset = bucketStart
	}
	if cmdLine >= m.detailScrollOffset+avail {
		m.detailScrollOffset = cmdLine - avail + 1
	}
}

// View implements tea.Model
func (m *Model) View() string {
	return m.renderView()
}

// Messages
type contextsLoadedMsg struct {
	contexts []ContextItem
}

type errMsg struct {
	err error
}

type commandContextLoadedMsg struct {
	before []models.Command
	target *models.Command
	after  []models.Command
}

type emptyStatePeeksMsg struct {
	prev *periodPeekData
	next *periodPeekData
}

// Getters for testing
func (m *Model) SelectedIdx() int {
	return m.selectedIdx
}

func (m *Model) Contexts() []ContextItem {
	return m.contexts
}

func (m *Model) CurrentDate() time.Time {
	return m.currentDate
}

func (m *Model) Focused() bool {
	return m.focused
}

func (m *Model) ViewState() ViewState {
	return m.viewState
}

func (m *Model) DetailBuckets() []DetailBucket {
	return m.detailBuckets
}

func (m *Model) DetailCommands() []models.Command {
	return m.detailCommands
}

func (m *Model) DetailCmdIdx() int {
	return m.detailCmdIdx
}

func (m *Model) DetailScrollOffset() int {
	return m.detailScrollOffset
}

func (m *Model) DisplayMode() DisplayMode {
	return m.displayMode
}

func (m *Model) CmdDetailTarget() *models.Command {
	if m.cmdDetailIdx < len(m.cmdDetailAll) {
		cmd := m.cmdDetailAll[m.cmdDetailIdx]
		return &cmd
	}
	return nil
}

func (m *Model) CmdDetailIdx() int {
	return m.cmdDetailIdx
}

func (m *Model) CmdDetailBefore() []models.Command {
	if m.cmdDetailStartIdx > 0 {
		return m.cmdDetailAll[:m.cmdDetailStartIdx]
	}
	return nil
}

func (m *Model) CmdDetailAfter() []models.Command {
	if m.cmdDetailStartIdx+1 < len(m.cmdDetailAll) {
		return m.cmdDetailAll[m.cmdDetailStartIdx+1:]
	}
	return nil
}

func (m *Model) CmdDetailTotalContext() int {
	return m.cmdDetailTotalContext()
}

func (m *Model) CmdDetailAll() []models.Command {
	return m.cmdDetailAll
}

func (m *Model) Period() Period {
	return m.period
}

func (m *Model) FilterText() string {
	return m.filterText
}

func (m *Model) FilterActive() bool {
	return m.filterActive
}

func (m *Model) EmptyPrevPeriod() *periodPeekData {
	return m.emptyPrevPeriod
}

func (m *Model) EmptyNextPeriod() *periodPeekData {
	return m.emptyNextPeriod
}

func (m *Model) HelpPreviousView() ViewState {
	return m.helpPreviousView
}

// filterBySubstring returns commands where CommandText contains the filter string
func filterBySubstring(commands []models.Command, filter string) []models.Command {
	if filter == "" {
		return commands
	}
	var result []models.Command
	for _, cmd := range commands {
		if strings.Contains(cmd.CommandText, filter) {
			result = append(result, cmd)
		}
	}
	return result
}

// commandFrequencies counts occurrences of each command text
func commandFrequencies(commands []models.Command) map[string]int {
	freq := make(map[string]int)
	for _, cmd := range commands {
		freq[cmd.CommandText]++
	}
	return freq
}

// filterByMode returns commands matching the mode
func filterByMode(commands []models.Command, mode DisplayMode) []models.Command {
	if mode == AllMode {
		return commands
	}
	freq := commandFrequencies(commands)
	var result []models.Command
	for _, cmd := range commands {
		count := freq[cmd.CommandText]
		if mode == UniqueMode && count == 1 {
			result = append(result, cmd)
		}
	}
	return result
}

// filteredCommandCount returns the count of commands matching the filter and mode
func filteredCommandCount(commands []models.Command, mode DisplayMode, filter string) int {
	filtered := filterBySubstring(commands, filter)
	return len(filterByMode(filtered, mode))
}
