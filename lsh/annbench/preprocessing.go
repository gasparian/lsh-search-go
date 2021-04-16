package annbench

import (
	"log"
	"runtime"
	// "unsafe"
	// "gonum.org/v1/hdf5"
)

// Objects inside the hdf5:
// train
// test
// distances
// neighbors

type FeatureVec [96]float32 // TODO: could be 784 for fashion mnist or 65 for glove
type NeighborsIds [100]int32
type DistanceVec [100]float32

// // GetVectorsFromHDF5 returns slice of feature vectors, from the hdf5 table
// func GetVectorsFromHDF5(table *hdf5.File, datasetName string) ([]FeatureVec, error) {
// 	dataset, err := table.OpenDataset(datasetName)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer dataset.Close()

// 	fileSpace := dataset.Space()
// 	numTicks := fileSpace.SimpleExtentNPoints()
// 	numTicks /= (int)(unsafe.Sizeof(FeatureVec{})) / 4

// 	ticks := make([]FeatureVec, numTicks)
// 	err = dataset.Read(&ticks)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return ticks, nil
// }

// // GetNeighborsFromHDF5 returns slice of feature vectors, from the hdf5 table
// func GetNeighborsFromHDF5(table *hdf5.File, datasetName string) ([]NeighborsIds, error) {
// 	dataset, err := table.OpenDataset(datasetName)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer dataset.Close()

// 	fileSpace := dataset.Space()
// 	numTicks := fileSpace.SimpleExtentNPoints()
// 	numTicks /= (int)(unsafe.Sizeof(NeighborsIds{})) / 4

// 	ticks := make([]NeighborsIds, numTicks)
// 	err = dataset.Read(&ticks)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return ticks, nil
// }

// PrintMemUsage __
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	log.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	log.Printf("\tSys = %v MiB", bToMb(m.Sys))
	log.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
