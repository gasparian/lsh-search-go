package annbench

import (
	"sort"
)

// Recall returns ratio of relevant predictions over the all true relevant items
// both arrays MUST BE SORTED
func Recall(prediction, groundTruth []int) float64 {
	valid := 0
	for _, val := range prediction {
		idx := sort.SearchInts(groundTruth, val)
		if idx < len(groundTruth) {
			valid++
		}
	}
	return float64(valid) / float64(len(groundTruth))
}
