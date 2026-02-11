package tui

import (
	"sort"
)

// sortContextItems sorts contexts by command count descending,
// then alphabetically by working dir and branch as a tiebreaker.
func sortContextItems(items []ContextItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].CommandCount != items[j].CommandCount {
			return items[i].CommandCount > items[j].CommandCount
		}
		if items[i].Key.WorkingDir != items[j].Key.WorkingDir {
			return items[i].Key.WorkingDir < items[j].Key.WorkingDir
		}
		return items[i].Branch < items[j].Branch
	})
}
