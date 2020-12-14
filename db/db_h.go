package db

import (
	"os"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	sampleSize, _ = strconv.Atoi(os.Getenv("SAMPLE_SIZE"))

	// GroupMeanStd holds pipeline for mongodb aggregation
	GroupMeanStd = mongo.Pipeline{
		bson.D{{"$sample", bson.D{
			{"size", sampleSize},
		}}},
		bson.D{{"$unwind", bson.D{
			{"path", "$featureVec"},
			{"includeArrayIndex", "i"},
		}}},
		bson.D{{"$group", bson.D{
			{"_id", "$i"},
			{"avg", bson.D{
				{"$avg", "$featureVec"},
			}},
			{"std", bson.D{
				{"$stdDevSamp", "$featureVec"},
			}},
		}}},
		bson.D{{"$sort", bson.D{
			{"_id", 1},
		}}},
		bson.D{{"$group", bson.D{
			{"_id", "null"},
			{"avg", bson.D{
				{"$push", "$avg"},
			}},
			{"std", bson.D{
				{"$push", "$std"},
			}},
		}}},
	}
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
	OrigID       int                `bson:"origId"`
	NeighborsIds []int32            `bson:"neighborsIds,omitempty"`
	FeatureVec   []float64          `bson:"featureVec,omitempty"` // TO DO: worth to use cm.Vector type
	Hashes       map[int32]uint64   `bson:"hashes,omitempty"`     // TO DO: the field should be deleted later
}

// HashesRecord stores the id of original document in other collection and hashes map
type HashesRecord struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	Hashes map[int]uint64     `bson:"hashes,omitempty"`
}

// HelperRecord holds the indexer model and supplementary data
// TO DO: split helper record on two documents, one those will be holding just the indexer object
type HelperRecord struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Indexer      []byte             `bson:"indexer,omitempty"`
	IsBuildDone  bool               `bson:"isBuildDone,omitempty"`
	HashCollName string             `bson:"hashCollName,omitempty"`
}

// MongoClient holds client for connecting to the mongodb
type MongoClient struct {
	Client *mongo.Client
}

// FindQuery needs to perform find operation with mongodb
type FindQuery struct {
	Limit int
	Proj  bson.M
	Query bson.D
}
