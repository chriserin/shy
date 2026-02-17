package tui

// helpBinding represents a single keybinding entry for the help view.
type helpBinding struct {
	key  string
	desc string
}

// bindingsForView returns the help bindings for the given view state.
func bindingsForView(vs ViewState) []helpBinding {
	switch vs {
	case ContextDetailView:
		return contextDetailBindings()
	case CommandDetailView:
		return commandDetailBindings()
	case CommandTextView:
		return commandTextBindings()
	default:
		return summaryBindings()
	}
}

func summaryBindings() []helpBinding {
	return []helpBinding{
		{"j", "Navigate down"},
		{"k", "Navigate up"},
		{"enter", "Open context"},
		{"h", "Previous period"},
		{"l", "Next period"},
		{"t", "Today"},
		{"e", "Yesterday"},
		{"u", "Unique mode"},
		{"a", "All mode"},
		{"/", "Filter"},
		{"esc", "Clear filter"},
		{"]", "Cycle period up"},
		{"[", "Cycle period down"},
		{"?", "Help"},
		{"q", "Quit"},
	}
}

func contextDetailBindings() []helpBinding {
	return []helpBinding{
		{"j", "Navigate down"},
		{"k", "Navigate up"},
		{"enter", "View command detail"},
		{"y", "Yank command"},
		{"S", "Star command"},
		{"D", "Delete command"},
		{"-", "Back to summary"},
		{"H", "Previous context"},
		{"L", "Next context"},
		{"h", "Previous period"},
		{"l", "Next period"},
		{"t", "Today"},
		{"e", "Yesterday"},
		{"u", "Unique mode"},
		{"a", "All mode"},
		{"/", "Filter"},
		{"esc", "Clear filter"},
		{"]", "Cycle period up"},
		{"[", "Cycle period down"},
		{"?", "Help"},
		{"q", "Quit"},
	}
}

func commandTextBindings() []helpBinding {
	return []helpBinding{
		{"j", "Scroll down"},
		{"k", "Scroll up"},
		{"y", "Yank command"},
		{"-", "Back"},
		{"?", "Help"},
		{"q", "Quit"},
	}
}

func commandDetailBindings() []helpBinding {
	return []helpBinding{
		{"j", "Navigate down"},
		{"k", "Navigate up"},
		{"y", "Yank command"},
		{"S", "Star command"},
		{"D", "Delete command"},
		{"-", "Back to context"},
		{"?", "Help"},
		{"q", "Quit"},
	}
}
