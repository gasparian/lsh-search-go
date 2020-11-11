package db

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Objects inside the hdf5:
// train
// test
// distances
// neighbors

type FeatureVec [96]float32
type NeighborsIds [100]int32
type DistanceVec [100]float32

// VectorRecord used to store the vectors to search in the mongodb
type VectorRecord struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	OrigID       int                `bson:"origId,omitempty"`
	NeighborsIds []int32            `bson:"neighborsIds,omitempty"`
	FeatureVec   []float64          `bson:"featureVec,omitempty"`
	Hashes       map[int32]uint64   `bson:"hashes,omitempty"`
}
