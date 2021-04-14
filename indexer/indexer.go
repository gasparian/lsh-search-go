// TODO:
// CreateCollection --> client.Create(bucketName string) err
// GetCollection --> client.Get(bucketName, key string) val, ok
// DropCollection --> client.Del(bucketName, key string) err / client.Destroy(bucketName string) err
// GetCollSize --> client.Size(bucketName string) size, err
//
// Newly added funcs:
// Calculate mean, std of random sample of records (e.g. from bucket with Train data)
// Populate collection with set of vectors (e.g. Train/Test with "original" vectors)

package indexer

import (
	// "errors"
	"os"
	// "time"

	// "context"
	// "fmt"
	// "sort"
	"strconv"

	cm "github.com/gasparian/lsh-search-service/common"
	hashing "github.com/gasparian/lsh-search-service/lsh"
)

// Used to represent the hasher build status
const (
	BuildStatusUnknown = iota
	BuildStatusError
	BuildStatusInProgress
	BuildStatusDone
)

// DbClient holds interface for the kv storage
type DbClient interface {
	New(string, int)
	Open() error
	Close() error
	Create(string) error
	Destroy(string) error
	Del(string, string) error
	Set(string, string, []byte) error
	Get(string, string) ([]byte, bool)
	MakeIterator(string) error
	Next(string) (string, []byte, error)
}

// NeighborsRecord holds a single neighbor
// Used only to store filtered neighbors for sorting
type NeighborsRecord struct {
	Key  string
	Dist float64
}

// DatasetStats holds basic feature vector stats like mean and standart deviation
type DatasetStats struct {
	Mean []float64
	Std  []float64
}

// VectorRecord (the same as RequestData?) used to store the vectors to search in the mongodb
type VectorRecord struct {
	Key       string
	Neighbors []uint64
	Vec       []float64
}

// HashRecord stores generated hash and a key of the original vector
type HashRecord struct {
	Key       string
	Hash      uint64
	VectorKey string
}

// HasherState holds the Hasher model and supplementary data
type HasherState struct {
	VectorsBucketKey string
	Hasher           []byte
	IsBuildDone      bool
	BuildError       string
	LastBuildTime    int64
	BuildElapsedTime int64
}

// Config holds all needed params for hasher
type Config struct {
	Hasher          hashing.Config
	DbAddress       string
	DbClientTimeout int
	BatchSize       int
	MaxHashesQuery  int
	MaxNN           int
}

// Indexer is a key struct that holds all data needed to work with search index
type Indexer struct {
	StateBucketName   string
	VectorsBucketName string
	Config            *Config
	Stats             DatasetStats
	State             HasherState
	Hasher            *hashing.Hasher
	Logger            *cm.Logger
	DbClient          DbClient
}

// ParseEnv forms app config by parsing the environment variables
func ParseEnv() (*Config, error) {
	intVars := make(map[string]int)
	intVarsNames := []string{
		"DB_CLIENT_TIMEOUT",
		"BATCH_SIZE",
		"MAX_HASHES_QUERY",
		"MAX_NN",
		"ANGULAR_METRIC",
		"N_PLANES",
		"N_PERMUTS",
		"BIAS_MULTIPLIER",
	}
	for _, key := range intVarsNames {
		val, err := strconv.Atoi(os.Getenv(key))
		if err != nil {
			return nil, err
		}
		intVars[key] = val
	}
	distanceThrsh, err := strconv.ParseFloat(os.Getenv("DISTANCE_THRSH"), 32)
	if err != nil {
		return nil, err
	}
	config := &Config{
		DbAddress:       os.Getenv("DB_ADDRESS"),
		DbClientTimeout: intVars["DB_CLIENT_TIMEOUT"],
		BatchSize:       intVars["BATCH_SIZE"],
		MaxHashesQuery:  intVars["MAX_HASHES_QUERY"],
		MaxNN:           intVars["MAX_NN"],
		Hasher: hashing.Config{
			IsAngularDistance: intVars["ANGULAR_METRIC"],
			NPlanes:           intVars["N_PLANES"],
			NPermutes:         intVars["N_PERMUTS"],
			BiasMultiplier:    float64(intVars["BIAS_MULTIPLIER"]),
			DistanceThrsh:     distanceThrsh,
		},
	}
	return config, nil
}

// New returns empty index object with initialized db client
func New(config *Config, logger *cm.Logger, dbClient DbClient) (*Indexer, error) {
	dbClient.New(config.DbAddress, config.DbClientTimeout)
	err := dbClient.Open()
	if err != nil {
		return nil, err
	}
	indexer := &Indexer{
		Config:   config,
		Logger:   logger,
		Hasher:   hashing.NewLSHIndex(config.Hasher),
		DbClient: dbClient,
	}
	err = indexer.LoadHasher() // TODO: implement
	if err != nil {
		logger.Err.Println("Loading Hasher object: " + err.Error())
		return nil, err
	}
	helperExists, err := annServer.Mongo.CheckCollection(config.Db.HelperCollectionName)
	if err != nil {
		logger.Err.Println("Checking helper collection: " + err.Error())
		return ANNServer{}, err
	}
	if !helperExists {
		_, err = annServer.Mongo.CreateCollection(config.Db.HelperCollectionName)
		if err != nil {
			logger.Err.Println("Creating helper collection: " + err.Error())
			return ANNServer{}, err
		}
	}
	return annServer, nil
}

// LoadHasher load Hasher from the db if it exists
func (annServer *ANNServer) LoadHasher() error {
	HasherRecord, err := annServer.GetHelperRecord(true)
	if err != nil {
		return err
	}
	if len(HasherRecord.Hasher) > 0 && HasherRecord.IsBuildDone {
		annServer.Hasher.Load(HasherRecord.Hasher)
		annServer.HashCollName = HasherRecord.HashCollName
	}
	return nil
}

// GetHelperRecord gets supplementary data from the specified collection
func (annServer *ANNServer) GetHelperRecord(getHasherObject bool) (storage.HelperRecord, error) {
	proj := bson.M{}
	if !getHasherObject {
		proj = bson.M{"Hasher": 0}
	}
	helperColl := annServer.Mongo.GetCollection(annServer.Mongo.Config.HelperCollectionName)
	cursor, err := helperColl.GetCursor(
		db.FindQuery{
			Limit: 1,
			Query: bson.D{
				{"Hasher", bson.D{{"$exists", true}}},
			},
			Proj: proj,
		},
	)
	if err != nil {
		return storage.HelperRecord{}, err
	}
	var results []storage.HelperRecord
	err = cursor.All(context.Background(), &results)
	if err != nil || len(results) != 1 {
		return storage.HelperRecord{}, err
	}
	return results[0], nil
}

// // UpdateBuildStatus updates helper record with the new biuld status and error
// func (annServer *ANNServer) UpdateBuildStatus(status storage.HelperRecord) error {
// 	helperColl := annServer.Mongo.GetCollection(annServer.Mongo.Config.HelperCollectionName)
// 	err := helperColl.UpdateField(
// 		bson.D{
// 			{"Hasher", bson.D{
// 				{"$exists", true},
// 			}}},
// 		bson.D{
// 			{"$set", bson.D{
// 				{"isBuildDone", status.IsBuildDone},
// 				{"buildError", status.BuildError},
// 				{"lastBuildTime", status.LastBuildTime},
// 				{"buildElapsedTime", status.BuildElapsedTime},
// 			}}})

// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// // hashBatch accumulates db documents in a batch of desired length and calculates hashes
// func (annServer *ANNServer) hashBatch(vecs []cm.RequestData) ([]interface{}, error) {
// 	batch := make([]interface{}, len(vecs))
// 	for idx, vec := range vecs {
// 		batch[idx] = storage.HashesRecord{
// 			SecondaryID: vec.SecondaryID,
// 			FeatureVec:  vec.Vec,
// 			Hashes:      annServer.Hasher.GetHashes(cm.NewVec(vec.Vec)),
// 		}
// 	}
// 	return batch, nil
// }

// // TryUpdateLocalHasher checks if there is a fresher build in db, and if it is - updates the local hasher
// func (annServer *ANNServer) TryUpdateLocalHasher() error {
// 	helperRecord, err := annServer.GetHelperRecord(false)
// 	if err != nil {
// 		return err
// 	}
// 	dt := helperRecord.LastBuildTime - annServer.LastBuildTime
// 	isBuildValid := helperRecord.IsBuildDone && len(helperRecord.BuildError) == 0
// 	if isBuildValid && dt > 0 {
// 		err = annServer.LoadHasher()
// 		if err != nil {
// 			return err
// 		}
// 	} else if !isBuildValid {
// 		return errors.New("build is in progress or not valid. Please, do not use the index right now")
// 	}
// 	return nil
// }

// // BuildIndex gets data stats from the db and creates the new Hasher (or hasher) object
// // and submits status to the helper collection
// func (annServer *ANNServer) BuildIndex(input cm.DatasetStats) error {
// 	start := time.Now().UnixNano()
// 	// NOTE: check if the previous build has been done
// 	helperRecord, err := annServer.GetHelperRecord(false)
// 	if err != nil {
// 		annServer.Logger.Warn.Println("Building index: seems like helper record does not exist yet")
// 	}
// 	if !helperRecord.IsBuildDone || len(helperRecord.BuildError) != 0 {
// 		return errors.New("Building index: aborting - previous build is not done yet")
// 	}

// 	err = annServer.UpdateBuildStatus(
// 		storage.HelperRecord{
// 			IsBuildDone: false,
// 		},
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	err = annServer.Hasher.Generate(cm.NewVec(input.Mean), cm.NewVec(input.Std))
// 	if err != nil {
// 		return err
// 	}
// 	annServer.Logger.Info.Println(annServer.Hasher.Instances[0]) // DEBUG - check for not being [0]

// 	lshSerialized, err := annServer.Hasher.Dump()
// 	if err != nil {
// 		return err
// 	}

// 	// NOTE: Getting old hash collection name
// 	oldHelperRecord, err := annServer.GetHelperRecord(false)
// 	if err != nil {
// 		return err
// 	}

// 	// NOTE: Generating and saving new hash collection, keeping the old one
// 	newHashCollName, err := cm.GetRandomID()
// 	if err != nil {
// 		return err
// 	}
// 	_, err = annServer.Mongo.CreateCollection(newHashCollName)
// 	if err != nil {
// 		return err
// 	}

// 	// NOTE: create indexes for the all new fields
// 	hashesColl := annServer.Mongo.GetCollection(newHashCollName)
// 	err = hashesColl.CreateIndexesByFields(annServer.Hasher.HashFieldsNames, false)
// 	if err != nil {
// 		return err
// 	}
// 	// NOTE: drop old collection with hashes
// 	if len(oldHelperRecord.HashCollName) != 0 {
// 		err = annServer.Mongo.DropCollection(oldHelperRecord.HashCollName)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	// NOTE: update helper with the new Hasher object and info
// 	helperColl := annServer.Mongo.GetCollection(annServer.Config.Db.HelperCollectionName)
// 	end := time.Now().UnixNano()
// 	annServer.LastBuildTime = end
// 	err = helperColl.UpdateField(
// 		bson.D{
// 			{"hasher", bson.D{
// 				{"$exists", true},
// 			}}},
// 		bson.D{
// 			{"$set", bson.D{
// 				{"isBuildDone", true},
// 				{"buildError", ""},
// 				{"hasher", lshSerialized},
// 				{"hashCollName", newHashCollName},
// 				{"lastBuildTime", end},
// 				{"buildElapsedTime", end - start},
// 			}}})

// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// // GetHashCollSize returns number of documents in hash collection
// func (annServer *ANNServer) GetHashCollSize() (int64, error) {
// 	err := annServer.TryUpdateLocalHasher()
// 	if err != nil {
// 		return 0, err
// 	}
// 	size, err := annServer.Mongo.GetCollSize(annServer.HashCollName)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return size, nil
// }

// // popHashRecord drops record from collection by SecondaryID (ID - is mongo-specific id)
// func (annServer *ANNServer) popHashRecord(id uint64) error {
// 	err := annServer.TryUpdateLocalHasher()
// 	if err != nil {
// 		return err
// 	}
// 	helperRecord, err := annServer.GetHelperRecord(false)
// 	if err != nil {
// 		return err
// 	}
// 	hashesColl := annServer.Mongo.GetCollection(helperRecord.HashCollName)
// 	err = hashesColl.DeleteRecords(bson.D{{"secondaryId", id}})
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// // putHashRecord drops record from collection by objectID (string Hex)
// func (annServer *ANNServer) putHashRecord(vecs []cm.RequestData) error {
// 	err := annServer.TryUpdateLocalHasher()
// 	if err != nil {
// 		return err
// 	}
// 	helperRecord, err := annServer.GetHelperRecord(false)
// 	if err != nil {
// 		return err
// 	}
// 	hashesColl := annServer.Mongo.GetCollection(helperRecord.HashCollName)
// 	records, err := annServer.hashBatch(vecs)
// 	if err != nil {
// 		return err
// 	}
// 	err = hashesColl.SetRecords(records)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// // getNeighbors returns filtered nearest neighbors sorted by distance in ascending order
// func (annServer *ANNServer) getNeighbors(input cm.RequestData) (*cm.ResponseData, error) {
// 	err := annServer.TryUpdateLocalHasher()
// 	if err != nil {
// 		return nil, err
// 	}
// 	helperRecord, err := annServer.GetHelperRecord(false)
// 	if err != nil {
// 		return nil, err
// 	}
// 	hashesColl := annServer.Mongo.GetCollection(helperRecord.HashCollName)
// 	inputVec := cm.NewVec(input.Vec)
// 	hashes := annServer.Hasher.GetHashes(inputVec)
// 	hashesQuery := bson.D{}
// 	for k, v := range hashes {
// 		hashesQuery = append(hashesQuery, bson.E{strconv.Itoa(k), v})
// 	}
// 	hashesCursor, err := hashesColl.GetCursor(
// 		db.FindQuery{
// 			Limit: annServer.Config.App.MaxHashesQuery,
// 			Query: hashesQuery,
// 			Proj:  bson.M{"_id": 1, "featureVec": 1},
// 		},
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var neighbors []cm.NeighborsRecord
// 	var idx int = 0
// 	var candidate storage.HashesRecord
// 	for hashesCursor.Next(context.Background()) {
// 		if err := hashesCursor.Decode(&candidate); err != nil {
// 			continue
// 		}
// 		dist, ok := annServer.Hasher.GetDist(inputVec, cm.NewVec(candidate.FeatureVec))
// 		if ok {
// 			neighbors = append(neighbors, cm.NeighborsRecord{
// 				SecondaryID: candidate.SecondaryID,
// 				Dist:        dist,
// 			})
// 			idx++
// 		}
// 	}
// 	sort.Slice(neighbors, func(i, j int) bool {
// 		return neighbors[i].Dist < neighbors[j].Dist
// 	})
// 	answerSize := annServer.Config.App.MaxNN
// 	if len(neighbors) < answerSize {
// 		answerSize = len(neighbors)
// 	}
// 	neighborsIDs := make([]uint64, answerSize)
// 	for i := 0; i < answerSize; i++ {
// 		neighborsIDs[i] = neighbors[i].SecondaryID
// 	}
// 	return &cm.ResponseData{
// 		Results: neighborsIDs,
// 	}, nil
// }
