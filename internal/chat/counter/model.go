package counter

import (
	"encoding/json"
)

type Counts struct {
	itemsAndCounts [][]interface{}
}

func (c *Counts) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.itemsAndCounts)
}

// Non-threadsafe - only share copies!
type frequencies struct {
	countsByItem map[string]int
	itemsByCount map[int][]string
}

// Mutates state
func (f *frequencies) update(item string, delta int) {
	if delta == 0 {
		return
	}

	oldCount := f.countsByItem[item]
	newCount := oldCount + delta

	f.countsByItem[item] = newCount

	// Add to new count items
	if delta > 0 {
		f.itemsByCount[newCount] = append(f.itemsByCount[newCount], item)
	} else {
		f.itemsByCount[newCount] = append([]string{item}, f.itemsByCount[newCount]...)
	}
	// Remove from old count items
	oldCountItems := make([]string, 0, len(f.itemsByCount[oldCount]))
	for _, oldCountItem := range f.itemsByCount[oldCount] {
		if item != oldCountItem {
			oldCountItems = append(oldCountItems, oldCountItem)
		}
	}
	f.itemsByCount[oldCount] = oldCountItems
}

func newFrequencies(initialCapacity int) frequencies {
	return frequencies{
		countsByItem: make(map[string]int, initialCapacity),
		itemsByCount: make(map[int][]string, initialCapacity),
	}
}
