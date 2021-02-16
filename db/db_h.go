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
	SecondaryID  uint64             `bson:"secondaryId"` // needed primarily for benchmarks
	NeighborsIds []uint64           `bson:"neighborsIds,omitempty"`
	FeatureVec   []float64          `bson:"featureVec,omitempty"`
}

// HashesRecord stores the id of original document in other collection and hashes map
type HashesRecord struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	SecondaryID uint64             `bson:"secondaryId,omitempty"`
	FeatureVec  []float64          `bson:"featureVec,omitempty"`
	Hashes      map[int]uint64     `bson:"hashes,omitempty"`
}

// HelperRecord holds the Hasher model and supplementary data
type HelperRecord struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	Hasher           []byte             `bson:"hasher,omitempty"`
	IsBuildDone      bool               `bson:"isBuildDone,omitempty"`
	BuildError       string             `bson:"buildError,omitempty"`
	HashCollName     string             `bson:"hashCollName,omitempty"`
	LastBuildTime    int64              `bson:"lastBuildTime,omitempty"`
	BuildElapsedTime int64              `bson:"buildElapsedTime,omitempty"`
}

// Config holds db address and entities names
type Config struct {
	DbLocation           string
	DbName               string
	HelperCollectionName string
}

// MongoCollection is just an alias to original mongo Collection,
// to be able to add custom methods there
type MongoCollection struct {
	*mongo.Collection
}

// MongoDatastore holds mongo client and the database object
type MongoDatastore struct {
	Config  Config
	db      *mongo.Database
	Session *mongo.Client
}

// FindQuery needs to perform find operation with mongodb
type FindQuery struct {
	Limit int
	Proj  bson.M
	Query bson.D
}
