package app

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"context"
	"fmt"
	"sort"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	cm "github.com/gasparian/lsh-search-service/common"
	"github.com/gasparian/lsh-search-service/db"
	hashing "github.com/gasparian/lsh-search-service/lsh"
)

// getHelpMessage forms a byte array contains message
func getHelloMessage() []byte {
	helloMessage := cm.ResponseData{
		Message: `{
		"methods": {
			"GET/POST": {
				"/build-index": "starts building search index from scratch",
				"/check-build": "returns current build status",
				"/pop-hash": "removes the point from the search index",
				"/put-hash": "adds the point to the search index"
			},
			"POST": {
				"/get-nn": "returns db ids and distances of the nearest data points"
			}
	    }
	}`}
	// NOTE: ugly, but it's more convinient to update the text message by hand and then serialize to json
	out, err := json.Marshal(helloMessage)
	if err != nil {
		return []byte("")
	}
	return out
}

// ParseEnv forms app config by parsing the environment variables
func ParseEnv() (*ServiceConfig, error) {
	intVars := map[string]int{
		"BATCH_SIZE":       1000,
		"MAX_HASHES_QUERY": 10000,
		"MAX_NN":           100,
		"ANGULAR_METRIC":   0,
		"N_PLANES":         30,
		"N_PERMUTS":        5,
		"BIAS_MULTIPLIER":  1,
	}
	for key := range intVars {
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
	stringVars := map[string]string{
		"MONGO_ADDR": "", "DB_NAME": "",
		"COLLECTION_NAME": "", "HELPER_COLLECTION_NAME": "",
	}
	for key := range stringVars {
		val := os.Getenv(key)
		if len(val) == 0 {
			return nil, fmt.Errorf("Env value can't be empty: %s", key)
		}
		stringVars[key] = val
	}

	config := &ServiceConfig{
		Db: db.Config{
			DbLocation:           stringVars["MONGO_ADDR"],
			DbName:               stringVars["DB_NAME"],
			HelperCollectionName: stringVars["HELPER_COLLECTION_NAME"],
		},
		App: Config{
			BatchSize:      intVars["BATCH_SIZE"],
			MaxHashesQuery: intVars["MAX_HASHES_QUERY"],
			MaxNN:          intVars["MAX_NN"],
		},
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

// NewANNServer returns empty index object with initialized mongo client
func NewANNServer(logger *cm.Logger, config *ServiceConfig) (ANNServer, error) {
	mongodb, err := db.New(config.Db)
	if err != nil {
		logger.Err.Println("Creating db client: " + err.Error())
		return ANNServer{}, err
	}

	annServer := ANNServer{
		Config: *config,
		Mongo:  *mongodb,
		Logger: logger,
		Hasher: hashing.NewLSHIndex(config.Hasher),
	}
	err = annServer.LoadHasher()
	if err != nil {
		logger.Err.Println("Loading Hasher object: " + err.Error())
		return ANNServer{}, err
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

// UpdateBuildStatus updates helper record with the new biuld status and error
func (annServer *ANNServer) UpdateBuildStatus(status db.HelperRecord) error {
	helperColl := annServer.Mongo.GetCollection(annServer.Mongo.Config.HelperCollectionName)
	err := helperColl.UpdateField(
		bson.D{
			{"Hasher", bson.D{
				{"$exists", true},
			}}},
		bson.D{
			{"$set", bson.D{
				{"isBuildDone", status.IsBuildDone},
				{"buildError", status.BuildError},
				{"lastBuildTime", status.LastBuildTime},
				{"buildElapsedTime", status.BuildElapsedTime},
			}}})

	if err != nil {
		return err
	}
	return nil
}

// GetHelperRecord gets supplementary data from the specified collection
func (annServer *ANNServer) GetHelperRecord(getHasherObject bool) (db.HelperRecord, error) {
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
		return db.HelperRecord{}, err
	}

	var results []db.HelperRecord
	err = cursor.All(context.Background(), &results)
	if err != nil || len(results) != 1 {
		return db.HelperRecord{}, err
	}
	return results[0], nil
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

// hashBatch accumulates db documents in a batch of desired length and calculates hashes
func (annServer *ANNServer) hashBatch(vecs []cm.RequestData) ([]interface{}, error) {
	batch := make([]interface{}, len(vecs))
	for idx, vec := range vecs {
		batch[idx] = db.HashesRecord{
			SecondaryID: vec.SecondaryID,
			FeatureVec:  vec.Vec,
			Hashes:      annServer.Hasher.GetHashes(cm.NewVec(vec.Vec)),
		}
	}
	return batch, nil
}

// TryUpdateLocalHasher checks if there is a fresher build in db, and if it is - updates the local hasher
func (annServer *ANNServer) TryUpdateLocalHasher() error {
	helperRecord, err := annServer.GetHelperRecord(false)
	if err != nil {
		return err
	}
	dt := helperRecord.LastBuildTime - annServer.LastBuildTime
	isBuildValid := helperRecord.IsBuildDone && len(helperRecord.BuildError) == 0
	if isBuildValid && dt > 0 {
		err = annServer.LoadHasher()
		if err != nil {
			return err
		}
	} else if !isBuildValid {
		return errors.New("build is in progress or not valid. Please, do not use the index right now")
	}
	return nil
}

// BuildIndex gets data stats from the db and creates the new Hasher (or hasher) object
// and submits status to the helper collection
func (annServer *ANNServer) BuildIndex(input cm.DatasetStats) error {
	start := time.Now().UnixNano()
	// NOTE: check if the previous build has been done
	helperRecord, err := annServer.GetHelperRecord(false)
	if err != nil {
		annServer.Logger.Warn.Println("Building index: seems like helper record does not exist yet")
	}
	if !helperRecord.IsBuildDone || len(helperRecord.BuildError) != 0 {
		return errors.New("Building index: aborting - previous build is not done yet")
	}

	err = annServer.UpdateBuildStatus(
		db.HelperRecord{
			IsBuildDone: false,
		},
	)
	if err != nil {
		return err
	}

	err = annServer.Hasher.Generate(cm.NewVec(input.Mean), cm.NewVec(input.Std))
	if err != nil {
		return err
	}
	annServer.Logger.Info.Println(annServer.Hasher.Instances[0]) // DEBUG - check for not being [0]

	lshSerialized, err := annServer.Hasher.Dump()
	if err != nil {
		return err
	}

	// NOTE: Getting old hash collection name
	oldHelperRecord, err := annServer.GetHelperRecord(false)
	if err != nil {
		return err
	}

	// NOTE: Generating and saving new hash collection, keeping the old one
	newHashCollName, err := cm.GetRandomID()
	if err != nil {
		return err
	}
	_, err = annServer.Mongo.CreateCollection(newHashCollName)
	if err != nil {
		return err
	}

	// NOTE: create indexes for the all new fields
	hashesColl := annServer.Mongo.GetCollection(newHashCollName)
	err = hashesColl.CreateIndexesByFields(annServer.Hasher.HashFieldsNames, false)
	if err != nil {
		return err
	}
	// NOTE: drop old collection with hashes
	if len(oldHelperRecord.HashCollName) != 0 {
		err = annServer.Mongo.DropCollection(oldHelperRecord.HashCollName)
		if err != nil {
			return err
		}
	}

	// NOTE: update helper with the new Hasher object and info
	helperColl := annServer.Mongo.GetCollection(annServer.Config.Db.HelperCollectionName)
	end := time.Now().UnixNano()
	annServer.LastBuildTime = end
	err = helperColl.UpdateField(
		bson.D{
			{"hasher", bson.D{
				{"$exists", true},
			}}},
		bson.D{
			{"$set", bson.D{
				{"isBuildDone", true},
				{"buildError", ""},
				{"hasher", lshSerialized},
				{"hashCollName", newHashCollName},
				{"lastBuildTime", end},
				{"buildElapsedTime", end - start},
			}}})

	if err != nil {
		return err
	}
	return nil
}

// GetHashCollSize returns number of documents in hash collection
func (annServer *ANNServer) GetHashCollSize() (int64, error) {
	err := annServer.TryUpdateLocalHasher()
	if err != nil {
		return 0, err
	}
	size, err := annServer.Mongo.GetCollSize(annServer.HashCollName)
	if err != nil {
		return 0, err
	}
	return size, nil
}

// popHashRecord drops record from collection by SecondaryID (ID - is mongo-specific id)
func (annServer *ANNServer) popHashRecord(id uint64) error {
	err := annServer.TryUpdateLocalHasher()
	if err != nil {
		return err
	}
	helperRecord, err := annServer.GetHelperRecord(false)
	if err != nil {
		return err
	}
	hashesColl := annServer.Mongo.GetCollection(helperRecord.HashCollName)
	err = hashesColl.DeleteRecords(bson.D{{"secondaryId", id}})
	if err != nil {
		return err
	}
	return nil
}

// putHashRecord drops record from collection by objectID (string Hex)
func (annServer *ANNServer) putHashRecord(vecs []cm.RequestData) error {
	err := annServer.TryUpdateLocalHasher()
	if err != nil {
		return err
	}
	helperRecord, err := annServer.GetHelperRecord(false)
	if err != nil {
		return err
	}
	hashesColl := annServer.Mongo.GetCollection(helperRecord.HashCollName)
	records, err := annServer.hashBatch(vecs)
	if err != nil {
		return err
	}
	err = hashesColl.SetRecords(records)
	if err != nil {
		return err
	}
	return nil
}

// getNeighbors returns filtered nearest neighbors sorted by distance in ascending order
func (annServer *ANNServer) getNeighbors(input cm.RequestData) (*cm.ResponseData, error) {
	err := annServer.TryUpdateLocalHasher()
	if err != nil {
		return nil, err
	}
	helperRecord, err := annServer.GetHelperRecord(false)
	if err != nil {
		return nil, err
	}
	hashesColl := annServer.Mongo.GetCollection(helperRecord.HashCollName)
	inputVec := cm.NewVec(input.Vec)
	hashes := annServer.Hasher.GetHashes(inputVec)
	hashesQuery := bson.D{}
	for k, v := range hashes {
		hashesQuery = append(hashesQuery, bson.E{strconv.Itoa(k), v})
	}
	hashesCursor, err := hashesColl.GetCursor(
		db.FindQuery{
			Limit: annServer.Config.App.MaxHashesQuery,
			Query: hashesQuery,
			Proj:  bson.M{"_id": 1, "featureVec": 1},
		},
	)
	if err != nil {
		return nil, err
	}

	var neighbors []cm.NeighborsRecord
	var idx int = 0
	var candidate db.HashesRecord
	for hashesCursor.Next(context.Background()) {
		if err := hashesCursor.Decode(&candidate); err != nil {
			continue
		}
		dist, ok := annServer.Hasher.GetDist(inputVec, cm.NewVec(candidate.FeatureVec))
		if ok {
			neighbors = append(neighbors, cm.NeighborsRecord{
				SecondaryID: candidate.SecondaryID,
				Dist:        dist,
			})
			idx++
		}
	}
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].Dist < neighbors[j].Dist
	})
	answerSize := annServer.Config.App.MaxNN
	if len(neighbors) < answerSize {
		answerSize = len(neighbors)
	}
	neighborsIDs := make([]uint64, answerSize)
	for i := 0; i < answerSize; i++ {
		neighborsIDs[i] = neighbors[i].SecondaryID
	}
	return &cm.ResponseData{
		Results: neighborsIDs,
	}, nil
}
