package annbench

// Recall returns ratio of relevant predictions over the true relevant items
func Recall(prediction, groundTruth []int32) float64 {
	if len(prediction) != len(groundTruth) {
		return 0.0
	}
	valid := 0
	for i := range prediction {
		if prediction[i] == groundTruth[i] {
			valid++
		}
	}
	return float64(valid) / float64(len(prediction))
}
