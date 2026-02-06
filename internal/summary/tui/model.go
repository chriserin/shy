package tui

import (
	"sort"
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
)

// DisplayMode controls which commands are shown based on frequency
type DisplayMode int

const (
	AllMode    DisplayMode = iota
	UniqueMode
	MultiMode
)

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
	viewState            ViewState
	detailBuckets        []DetailBucket
	detailCommands       []models.Command
	detailCmdIdx         int
	detailScrollOffset   int
	detailContextKey     summary.ContextKey
	detailContextBranch  summary.BranchKey
	pendingDetailReentry bool

	// Display mode
	displayMode       DisplayMode
	detailFrequencies map[string]int // command text → count across entire context

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

// dateRange returns the start and end timestamps for the current date
func (m *Model) dateRange() (int64, int64) {
	year, month, day := m.currentDate.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	endOfDay := startOfDay.AddDate(0, 0, 1)
	return startOfDay.Unix(), endOfDay.Unix()
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
					m.enterDetailView()
					found = true
					break
				}
			}
			if !found {
				// Context not on this day — show empty detail view
				m.viewState = ContextDetailView
				m.detailBuckets = nil
				m.detailCommands = nil
				m.detailCmdIdx = 0
				m.detailScrollOffset = 0
			}
		}
		return m, nil

	case errMsg:
		// TODO: handle error display
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch m.viewState {
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
			m.enterDetailView()
		}
		return m, nil

	case "h":
		m.currentDate = m.currentDate.AddDate(0, 0, -1)
		m.selectedIdx = 0
		return m, m.loadContexts

	case "l":
		now := m.now()
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		currentStart := time.Date(m.currentDate.Year(), m.currentDate.Month(), m.currentDate.Day(), 0, 0, 0, 0, time.Local)
		if !currentStart.Before(todayStart) {
			return m, nil
		}
		m.currentDate = m.currentDate.AddDate(0, 0, 1)
		m.selectedIdx = 0
		return m, m.loadContexts

	case "t":
		m.currentDate = m.now()
		m.selectedIdx = 0
		return m, m.loadContexts

	case "y":
		m.currentDate = m.now().AddDate(0, 0, -1)
		m.selectedIdx = 0
		return m, m.loadContexts

	case "u":
		m.displayMode = UniqueMode
		return m, nil

	case "m":
		m.displayMode = MultiMode
		return m, nil

	case "a":
		m.displayMode = AllMode
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

	case "esc", "-":
		m.viewState = SummaryView
		return m, nil

	case "H":
		// Switch to previous context
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m.enterDetailView()
		}
		return m, nil

	case "L":
		// Switch to next context
		if m.selectedIdx < len(m.contexts)-1 {
			m.selectedIdx++
			m.enterDetailView()
		}
		return m, nil

	case "h":
		m.currentDate = m.currentDate.AddDate(0, 0, -1)
		m.pendingDetailReentry = true
		return m, m.loadContexts

	case "l":
		now := m.now()
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		currentStart := time.Date(m.currentDate.Year(), m.currentDate.Month(), m.currentDate.Day(), 0, 0, 0, 0, time.Local)
		if !currentStart.Before(todayStart) {
			return m, nil
		}
		m.currentDate = m.currentDate.AddDate(0, 0, 1)
		m.pendingDetailReentry = true
		return m, m.loadContexts

	case "t":
		m.currentDate = m.now()
		m.selectedIdx = 0
		m.viewState = SummaryView
		return m, m.loadContexts

	case "y":
		m.currentDate = m.now().AddDate(0, 0, -1)
		m.selectedIdx = 0
		m.viewState = SummaryView
		return m, m.loadContexts

	case "u":
		m.displayMode = UniqueMode
		m.enterDetailView()
		return m, nil

	case "m":
		m.displayMode = MultiMode
		m.enterDetailView()
		return m, nil

	case "a":
		m.displayMode = AllMode
		m.enterDetailView()
		return m, nil
	}

	return m, nil
}

func (m *Model) enterDetailView() {
	ctx := m.contexts[m.selectedIdx]

	m.detailContextKey = ctx.Key
	m.detailContextBranch = ctx.Branch

	// Compute frequencies across entire context, then filter
	m.detailFrequencies = commandFrequencies(ctx.Commands)
	filtered := filterByMode(ctx.Commands, m.displayMode)

	// Bucket filtered commands by hour
	bucketMap := summary.BucketBy(filtered, summary.Hourly)
	orderedIDs := summary.GetOrderedBuckets(bucketMap)

	// Build detail buckets and flat command list
	var buckets []DetailBucket
	var flatCommands []models.Command

	for _, id := range orderedIDs {
		bucket := bucketMap[id]
		label := summary.FormatHour(id)

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
	avail := m.height - 5
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
		} else if mode == MultiMode && count > 1 {
			result = append(result, cmd)
		}
	}
	return result
}

// filteredCommandCount returns the count of commands matching the mode
func filteredCommandCount(commands []models.Command, mode DisplayMode) int {
	return len(filterByMode(commands, mode))
}
