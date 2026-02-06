package tui

import (
	"sort"
)

// sortContextItems sorts contexts by command count descending
func sortContextItems(items []ContextItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].CommandCount > items[j].CommandCount
	})
}
