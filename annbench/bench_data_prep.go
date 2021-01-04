package annbench

import (
	"context"
	"log"
	"runtime"
	"time"
	"unsafe"

	"gonum.org/v1/hdf5"
	"lsh-search-service/db"
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

// UploadDatasetMongoDb sends batches of provided data to the mongodb
func UploadDatasetMongoDb(collection db.MongoCollection, data []db.FeatureVec, neighbors []db.NeighborsIds, batchSize int) error {
	var batch []interface{} = nil
	dataLen := len(data)
	neighborsLen := len(neighbors)
	var batchIdx int = 0
	var tmpRecord db.VectorRecord
	for idx := range data {
		tmpRecord = db.VectorRecord{
			SecondaryID: uint64(idx),
			FeatureVec:  make([]float64, len(data[0])),
		}
		for valIdx := range data[idx] {
			tmpRecord.FeatureVec[valIdx] = float64(data[idx][valIdx])
		}
		if dataLen == neighborsLen {
			tmpRecord.NeighborsIds = make([]uint64, len(neighbors[0]))
			for valIdx := range neighbors[idx] {
				tmpRecord.NeighborsIds[valIdx] = uint64(neighbors[idx][valIdx])
			}
		}
		batch = append(batch, tmpRecord)

		if batchIdx == batchSize-1 || idx == dataLen-1 {
			_, err := collection.InsertMany(context.Background(), batch)
			if err != nil {
				return err
			}
			batchIdx = 0
			batch = nil

			time.Sleep(time.Millisecond * 50)
			runtime.GC()
			// printMemUsage()
		} else {
			batchIdx++
		}
	}
	return nil
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
