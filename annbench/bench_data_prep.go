package annbench

import (
	"log"
	"runtime"
	"unsafe"

	"github.com/gasparian/lsh-search-service/db"
	"gonum.org/v1/hdf5"
)

// GetVectorsFromHDF5 returns slice of feature vectors, from the hdf5 table
func GetVectorsFromHDF5(table *hdf5.File, datasetName string) ([]db.FeatureVec, error) {
	dataset, err := table.OpenDataset(datasetName)
	if err != nil {
		return nil, err
	}
	defer dataset.Close()

	fileSpace := dataset.Space()
	numTicks := fileSpace.SimpleExtentNPoints()
	numTicks /= (int)(unsafe.Sizeof(db.FeatureVec{})) / 4

	ticks := make([]db.FeatureVec, numTicks)
	err = dataset.Read(&ticks)
	if err != nil {
		return nil, err
	}
	return ticks, nil
}

// GetNeighborsFromHDF5 returns slice of feature vectors, from the hdf5 table
func GetNeighborsFromHDF5(table *hdf5.File, datasetName string) ([]db.NeighborsIds, error) {
	dataset, err := table.OpenDataset(datasetName)
	if err != nil {
		return nil, err
	}
	defer dataset.Close()

	fileSpace := dataset.Space()
	numTicks := fileSpace.SimpleExtentNPoints()
	numTicks /= (int)(unsafe.Sizeof(db.NeighborsIds{})) / 4

	ticks := make([]db.NeighborsIds, numTicks)
	err = dataset.Read(&ticks)
	if err != nil {
		return nil, err
	}
	return ticks, nil
}

func printMemUsage() {
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
