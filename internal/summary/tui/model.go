package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chris/shy/internal/db"
	"github.com/chris/shy/internal/summary"
)

// ContextItem represents a context with its command count
type ContextItem struct {
	Key          summary.ContextKey
	Branch       summary.BranchKey
	CommandCount int
}

// Model represents the TUI state
type Model struct {
	// Database
	db     *db.DB
	dbPath string

	// Data
	contexts    []ContextItem
	currentDate time.Time

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
		return m, nil

	case errMsg:
		// TODO: handle error display
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
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

	case "h":
		// Previous day
		m.currentDate = m.currentDate.AddDate(0, 0, -1)
		m.selectedIdx = 0
		return m, m.loadContexts

	case "l":
		// Next day (but not past today)
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
		// Jump to today
		m.currentDate = m.now()
		m.selectedIdx = 0
		return m, m.loadContexts

	case "y":
		// Jump to yesterday
		m.currentDate = m.now().AddDate(0, 0, -1)
		m.selectedIdx = 0
		return m, m.loadContexts
	}

	return m, nil
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
