package app

import (
	"encoding/json"
	"errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
	"time"

	"context"
	"fmt"
	"sort"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	cm "lsh-search-engine/common"
	"lsh-search-engine/db"
	hashing "lsh-search-engine/lsh"
)

// getHelpMessage forms a byte array contains message
func getHelloMessage() []byte {
	helloMessage := []byte(`{
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
	}`)
	// NOTE: ugly, but it's more convinient to update the text message by hand and then serialize to json
	var raw map[string]interface{}
	err := json.Unmarshal(helloMessage, &raw)
	out, _ := json.Marshal(raw)
	if err != nil {
		return []byte("")
	}
	return out
}

// ParseEnv forms app config by parsing the environment variables
func ParseEnv() (*ServiceConfig, error) {
	intVars := map[string]int{
		"BATCH_SIZE":       0,
		"MAX_HASHES_QUERY": 0,
		"MAX_NN":           0,
		"ANGULAR_METRIC":   0,
		"MAX_N_PLANES":     0,
		"N_PERMUTS":        0,
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
			DataCollectionName:   stringVars["COLLECTION_NAME"],
			HelperCollectionName: stringVars["HELPER_COLLECTION_NAME"],
		},
		App: Config{
			BatchSize:       intVars["BATCH_SIZE"],
			MaxHashesNumber: intVars["MAX_HASHES_QUERY"],
			MaxNN:           intVars["MAX_NN"],
			DistanceThrsh:   distanceThrsh,
		},
		Hasher: hashing.Config{
			IsAngularDistance: intVars["ANGULAR_METRIC"],
			MaxNPlanes:        intVars["MAX_N_PLANES"],
			NPermutes:         intVars["N_PERMUTS"],
		},
	}

	return config, nil
}

// NewANNServer returns empty index object with initialized mongo client
func NewANNServer(logger *cm.Logger, config *ServiceConfig) (ANNServer, error) {
	mongodb, err := db.GetDbClient(config.Db)
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

// LoadHasher load Hasher from the db if it exists
func (annServer *ANNServer) LoadHasher() error {
	HasherRecord, err := annServer.Mongo.GetHelperRecord(true)
	if err != nil {
		return err
	}
	if len(HasherRecord.Hasher) > 0 && HasherRecord.IsBuildDone {
		annServer.Hasher.Load(HasherRecord.Hasher)
	}
	return nil
}

// hashDbRecordsBatch accumulates db documents in a batch of desired length and calculates hashes
func (annServer *ANNServer) hashDbRecordsBatch(cursor *mongo.Cursor, batchSize int) ([]interface{}, error) {
	batch := make([]interface{}, batchSize)
	batchID := 0
	for cursor.Next(context.Background()) {
		var record db.VectorRecord
		if err := cursor.Decode(&record); err != nil {
			continue
		}
		hashes, err := annServer.Hasher.GetHashes(cm.Vector(record.FeatureVec))
		if err != nil {
			return nil, err
		}
		batch[batchID] = db.HashesRecord{
			ID:     record.ID,
			Hashes: hashes,
		}
		batchID++
	}
	return batch[:batchID], nil
}

// TryUpdateLocalHasher checks if there is a fresher build in db, and if it is - updates the local hasher
func (annServer *ANNServer) TryUpdateLocalHasher() error {
	helperRecord, err := annServer.Mongo.GetHelperRecord(false)
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
// and submitting status to the helper collection
func (annServer *ANNServer) BuildIndex() error {
	start := time.Now().UnixNano()
	// NOTE: check if the previous build has been done
	helperRecord, err := annServer.Mongo.GetHelperRecord(false)
	if err != nil {
		annServer.Logger.Warn.Println("Building index: seems like helper record does not exist yet")
	}
	if !helperRecord.IsBuildDone || len(helperRecord.BuildError) != 0 {
		return errors.New("Building index: aborting - previous build is not done yet")
	}

	err = annServer.Mongo.UpdateBuildStatus(
		db.HelperRecord{
			IsBuildDone: false,
		},
	)
	if err != nil {
		return err
	}

	dataColl := annServer.Mongo.GetCollection(annServer.Config.Db.DataCollectionName)
	convMean, convStd, err := dataColl.GetAggregatedStats()
	if err != nil {
		return err
	}

	annServer.Logger.Info.Println(convMean) // DEBUG - check for not being [0]
	annServer.Logger.Info.Println(convStd)  // DEBUG - check for not being [0]

	err = annServer.Hasher.Generate(convMean, convStd)
	if err != nil {
		return err
	}
	annServer.Logger.Info.Println(annServer.Hasher.Instances[0]) // DEBUG - check for not being [0]

	lshSerialized, err := annServer.Hasher.Dump()
	if err != nil {
		return err
	}

	// NOTE: Getting old hash collection name
	oldHelperRecord, err := annServer.Mongo.GetHelperRecord(false)
	if err != nil {
		return err
	}

	// NOTE: Generating and saving new hash collection, keeping the old one
	newHashCollName, err := cm.GetRandomID()
	if err != nil {
		return err
	}
	newHashColl, err := annServer.Mongo.CreateCollection(newHashCollName)
	if err != nil {
		return err
	}

	// NOTE: fill the new collection with pointers to documents (_id) and fields with hashes
	cursor, err := dataColl.GetCursor(
		db.FindQuery{
			Limit: 0,
			Query: bson.D{},
		},
	)
	for cursor.Next(context.Background()) {
		hashesBatch, err := annServer.hashDbRecordsBatch(cursor, annServer.Config.App.BatchSize)
		if err != nil {
			return err
		}
		err = newHashColl.SetRecords(hashesBatch)
		if err != nil {
			return err
		}
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

// popHashRecord drops record from collection by objectID (string Hex)
func (annServer *ANNServer) popHashRecord(id string) error {
	err := annServer.TryUpdateLocalHasher()
	if err != nil {
		return err
	}
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	helperRecord, err := annServer.Mongo.GetHelperRecord(false)
	if err != nil {
		return err
	}
	hashesColl := annServer.Mongo.GetCollection(helperRecord.HashCollName)
	err = hashesColl.DeleteRecords(bson.D{{"_id", objectID}})
	if err != nil {
		return err
	}
	return nil
}

// putHashRecord drops record from collection by objectID (string Hex)
func (annServer *ANNServer) putHashRecord(id string) error {
	err := annServer.TryUpdateLocalHasher()
	if err != nil {
		return err
	}
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	helperRecord, err := annServer.Mongo.GetHelperRecord(false)
	if err != nil {
		return err
	}
	hashesColl := annServer.Mongo.GetCollection(helperRecord.HashCollName)
	dataColl := annServer.Mongo.GetCollection(annServer.Config.Db.DataCollectionName)
	records, err := dataColl.GetDbRecords(
		db.FindQuery{
			Limit: 1,
			Query: bson.D{{"_id", objectID}},
		},
	)
	if err != nil {
		return err
	}
	recordInterfaces := make([]interface{}, len(records))
	for i := range records {
		recordInterfaces[i] = records[i]
	}
	err = hashesColl.SetRecords(recordInterfaces)
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
	helperRecord, err := annServer.Mongo.GetHelperRecord(false)
	if err != nil {
		return nil, err
	}
	hashesColl := annServer.Mongo.GetCollection(helperRecord.HashCollName)
	var hashesRecords []db.HashesRecord
	if len(input.ID) > 0 {
		objectID, err := primitive.ObjectIDFromHex(input.ID)
		if err != nil {
			return nil, err
		}
		dbRecords, err := hashesColl.GetDbRecords(
			db.FindQuery{
				Limit: 1,
				Query: bson.D{{"_id", objectID}},
				Proj:  bson.M{"featureVec": 1},
			},
		)
		if err != nil {
			return nil, err
		}
		if len(dbRecords) != 1 {
			return nil, errors.New("id must be presented in the database")
		}
		input.Vec = dbRecords[0].FeatureVec
	}

	inputVec := cm.Vector(input.Vec)
	hashes, err := annServer.Hasher.GetHashes(inputVec)
	hashesQuery := bson.D{}
	for k, v := range hashes {
		hashesQuery = append(hashesQuery, bson.E{strconv.Itoa(k), v})
	}
	hashesRecords, err = hashesColl.GetHashesRecords(
		db.FindQuery{
			Limit: annServer.Config.App.MaxHashesNumber,
			Query: hashesQuery,
			Proj:  bson.M{"_id": 1},
		},
	)
	if err != nil {
		return nil, err
	}

	vectorIDs := bson.A{}
	for idx := range hashesRecords {
		vectorIDs = append(vectorIDs, hashesRecords[idx].ID)
	}
	hashesRecords = nil

	dataColl := annServer.Mongo.GetCollection(annServer.Config.Db.DataCollectionName)
	vectorsCursor, err := dataColl.GetCursor(
		db.FindQuery{
			Query: bson.D{{"_id", bson.D{{"$in", vectorIDs}}}},
			Proj:  bson.M{"_id": 1, "featureVec": 1},
		},
	)
	if err != nil {
		return nil, err
	}

	var neighbors []cm.ResponseRecord
	var idx int = 0
	var candidate db.VectorRecord
	// TO DO: add batch processing in separate goroutines
	for vectorsCursor.Next(context.Background()) {
		if err := vectorsCursor.Decode(&candidate); err != nil {
			continue
		}
		hexID := candidate.ID.Hex()
		dist := annServer.Hasher.GetDist(inputVec, cm.Vector(candidate.FeatureVec))
		if dist <= annServer.Config.App.DistanceThrsh {
			neighbors = append(neighbors, cm.ResponseRecord{
				ID:     hexID,
				OrigID: candidate.OrigID,
				Dist:   dist,
			})
			idx++
		}
	}
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].Dist < neighbors[j].Dist
	})
	return &cm.ResponseData{Neighbors: neighbors[:annServer.Config.App.MaxNN]}, nil
}
