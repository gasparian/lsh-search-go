package db

import (
	"context"
	"unsafe"

	"go.mongodb.org/mongo-driver/mongo"
	"gonum.org/v1/hdf5"
)

// GetVectorsFromHDF5 returns slice of feature vectors, from the hdf5 table
func GetVectorsFromHDF5(table *hdf5.File, datasetName string) ([]FeatureVec, error) {
	dataset, err := table.OpenDataset(datasetName)
	if err != nil {
		return nil, err
	}
	defer dataset.Close()

	fileSpace := dataset.Space()
	numTicks := fileSpace.SimpleExtentNPoints()
	numTicks /= (int)(unsafe.Sizeof(FeatureVec{})) / 4

	ticks := make([]FeatureVec, numTicks)
	err = dataset.Read(&ticks)
	if err != nil {
		return nil, err
	}
	return ticks, nil
}

// GetNeighborsFromHDF5 returns slice of feature vectors, from the hdf5 table
func GetNeighborsFromHDF5(table *hdf5.File, datasetName string) ([]NeighborsIds, error) {
	dataset, err := table.OpenDataset(datasetName)
	if err != nil {
		return nil, err
	}
	defer dataset.Close()

	fileSpace := dataset.Space()
	numTicks := fileSpace.SimpleExtentNPoints()
	numTicks /= (int)(unsafe.Sizeof(NeighborsIds{})) / 4

	ticks := make([]NeighborsIds, numTicks)
	err = dataset.Read(&ticks)
	if err != nil {
		return nil, err
	}
	return ticks, nil
}

// SetDataMongoDb sends batches of provided data to the mongodb
func LoadDatasetMongoDb(collection *mongo.Collection, data []FeatureVec, neighbors []NeighborsIds, batchSize int) error {
	batch := make([]interface{}, batchSize)
	dataLen := len(data)
	neighborsLen := len(neighbors)
	var batchIdx int = 0
	var tmpRecord VectorRecord
	for idx := range data {
		tmpRecord = VectorRecord{
			OrigID:     idx,
			FeatureVec: make([]float64, len(data[0])),
		}
		for valIdx := range data[idx] {
			tmpRecord.FeatureVec[valIdx] = float64(data[idx][valIdx])
		}
		if dataLen == neighborsLen {
			tmpRecord.NeighborsIds = make([]int32, len(neighbors[0]))
			for valIdx := range neighbors[idx] {
				tmpRecord.NeighborsIds[valIdx] = neighbors[idx][valIdx]
			}
		}
		batch[batchIdx] = tmpRecord

		if batchIdx == batchSize-1 || idx == dataLen-1 {
			_, err := collection.InsertMany(context.TODO(), batch[:batchIdx+1])
			if err != nil {
				return err
			}
			batch = make([]interface{}, batchSize)
			batchIdx = 0
		} else {
			batchIdx++
		}
	}
	return nil
}
