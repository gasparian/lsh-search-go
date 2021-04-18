package annbench

import (
	"gonum.org/v1/hdf5"
	"log"
	"runtime"
)

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
