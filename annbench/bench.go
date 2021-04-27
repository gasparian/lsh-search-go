package annbench

import (
	"sort"
)

// Recall returns ratio of relevant predictions over the all true relevant items
// both arrays MUST BE SORTED
func PrecisionRecall(prediction, groundTruth []int) (float64, float64) {
	valid := 0
	for _, val := range prediction {
		idx := sort.SearchInts(groundTruth, val)
		if idx < len(groundTruth) {
			valid++
		}
	}
	precision := 0.0
	if len(prediction) > 0 {
		precision = float64(valid) / float64(len(prediction))
	}
	recall := float64(valid) / float64(len(groundTruth))
	return precision, recall
}
