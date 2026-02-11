package summary

// ContextSummary represents aggregated command data for a specific context
// A context is defined by the combination of working directory and git branch
type ContextSummary struct {
	WorkingDir   string
	GitBranch    *string // nil for non-git directories
	CommandCount int
	FirstTime    int64 // Unix timestamp
	LastTime     int64 // Unix timestamp
}
