package counter

import (
	"encoding/json"
)

type Counts struct {
	TokensAndCounts [][]any `json:"tokensAndCounts"` // Inner array is (string, int) pair
}

func (c *Counts) MarshalJSON() ([]byte, error) {
	return json.Marshal(c)
}

// Non-threadsafe - only share copies!
type multiSet[T comparable] struct {
	countsByElement map[T]int
	elementsByCount map[int][]T
}

// Mutates state
func (f *multiSet[T]) update(element T, delta int) {
	if delta == 0 {
		return
	}

	oldCount := f.countsByElement[element]
	newCount := oldCount + delta

	if newCount > 0 {
		f.countsByElement[element] = newCount

		// Add to new count elements
		if delta > 0 {
			f.elementsByCount[newCount] = append(f.elementsByCount[newCount], element)
		} else {
			f.elementsByCount[newCount] = append([]T{element}, f.elementsByCount[newCount]...)
		}
	} else {
		delete(f.countsByElement, element)
	}
	// Remove from old count elements
	oldCountElems := make([]T, 0, len(f.elementsByCount[oldCount]))
	for _, oldCountItem := range f.elementsByCount[oldCount] {
		if element != oldCountItem {
			oldCountElems = append(oldCountElems, oldCountItem)
		}
	}
	f.elementsByCount[oldCount] = oldCountElems
}

func newMultiSet[T comparable](initialCapacity int) multiSet[T] {
	return multiSet[T]{
		countsByElement: make(map[T]int, initialCapacity),
		elementsByCount: make(map[int][]T, initialCapacity),
	}
}
