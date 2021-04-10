package db

import (
	"os"
	"strconv"
)

var (
	sampleSize, _         = strconv.Atoi(os.Getenv("SAMPLE_SIZE"))
	dbtimeOut, _          = strconv.Atoi(os.Getenv("DB_CLIENT_TIMEOUT"))
	createIndexMaxTime, _ = strconv.Atoi(os.Getenv("CREATE_INDEX_MAX_TIME"))
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
	ID           uint64
	NeighborsIds []uint64
	FeatureVec   []float64
}

// HashesRecord stores the id of original document in other collection and hashes map
type HashesRecord struct {
	ID         uint64
	FeatureVec []float64
	Hashes     map[int]uint64
}

// HelperRecord holds the Hasher model and supplementary data
type HelperRecord struct {
	Hasher           []byte
	IsBuildDone      bool
	BuildError       string
	HashCollName     string
	LastBuildTime    int64
	BuildElapsedTime int64
}

// Config holds db address and entities names
type Config struct {
	DbLocation string
	DbName     string
}
