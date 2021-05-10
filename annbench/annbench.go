package main

import (
	"gonum.org/v1/hdf5"
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

// Objects inside the hdf5:
// train
// test
// distances
// neighbors

// GetVectorsFromHDF5 returns slice of feature vectors, from the hdf5 table
func GetVectorsFromHDF5(table *hdf5.File, datasetName string, vecs interface{}) error {
	dataset, err := table.OpenDataset(datasetName)
	if err != nil {
		return err
	}
	defer dataset.Close()

	fileSpace := dataset.Space()
	numTicks := fileSpace.SimpleExtentNPoints()

	switch vecs := vecs.(type) {
	case *[]float32:
		*vecs = make([]float32, numTicks)
	case *[]int32:
		*vecs = make([]int32, numTicks)
	}

	err = dataset.Read(vecs)
	if err != nil {
		return err
	}

	return nil
}
