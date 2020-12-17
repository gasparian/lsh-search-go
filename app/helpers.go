package app

import (
	"encoding/json"
	"errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"os"

	"context"
	"fmt"
	"log"
	"sort"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	cm "vector-search-go/common"
	"vector-search-go/db"
	hashing "vector-search-go/lsh"
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

// GetNewLoggers creates an instance of all needed loggers
func GetNewLoggers() Logger {
	return Logger{
		Warn:  log.New(os.Stderr, "[ warn  ]", log.LstdFlags|log.Lshortfile),
		Info:  log.New(os.Stderr, "[ info  ]", log.LstdFlags|log.Lshortfile),
		Build: log.New(os.Stderr, "[ build ]", log.LstdFlags|log.Lshortfile),
		Err:   log.New(os.Stderr, "[ error ]", log.LstdFlags|log.Lshortfile),
	}
}

// ParseEnv forms app config by parsing the environment variables
func ParseEnv() (Config, error) {
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
			return Config{}, err
		}
		intVars[key] = val
	}
	distanceThrsh, err := strconv.ParseFloat(os.Getenv("DISTANCE_THRSH"), 32)
	if err != nil {
		return Config{}, err
	}
	stringVars := map[string]string{
		"MONGO_ADDR": "", "DB_NAME": "",
		"COLLECTION_NAME": "", "HELPER_COLLECTION_NAME": "",
	}
	for key := range stringVars {
		val := os.Getenv(key)
		if val == "" {
			return Config{}, fmt.Errorf("Env value can't be empty: %s", key)
		}
		stringVars[key] = val
	}

	config := Config{
		App: AppConfig{
			DbLocation:           stringVars["MONGO_ADDR"],
			DbName:               stringVars["DB_NAME"],
			DataCollectionName:   stringVars["COLLECTION_NAME"],
			HelperCollectionName: stringVars["HELPER_COLLECTION_NAME"],
			BatchSize:            intVars["BATCH_SIZE"],
			MaxHashesNumber:      intVars["MAX_HASHES_QUERY"],
			MaxNN:                intVars["MAX_NN"],
			DistanceThrsh:        distanceThrsh,
		},
		Hasher: hashing.LSHConfig{
			IsAngularDistance: intVars["ANGULAR_METRIC"],
			MaxNPlanes:        intVars["MAX_N_PLANES"],
			NPermutes:         intVars["N_PERMUTS"],
		},
	}

	return config, nil
}

// NewANNServer returns empty index object with initialized mongo client
func NewANNServer(logger Logger, config Config) (ANNServer, error) {
	mongodb, err := db.GetDbClient(config.App.DbLocation)
	if err != nil {
		logger.Err.Println("Creating db client: " + err.Error())
		return ANNServer{}, err
	}

	searchHandler := ANNServer{
		Config:      config.App,
		MongoClient: *mongodb,
		Logger:      logger,
		Index:       hashing.NewLSHIndex(config.Hasher),
	}
	err = searchHandler.LoadIndexer()
	if err != nil {
		logger.Err.Println("Loading indexer object: " + err.Error())
		return ANNServer{}, err
	}
	return searchHandler, nil
}

// LoadIndexer load indexer from the db if it exists
func (annServer *ANNServer) LoadIndexer() error {
	database := annServer.MongoClient.GetDb(annServer.Config.DbName)
	helperColl := database.Collection(annServer.Config.HelperCollectionName)
	indexerRecord, err := db.GetHelperRecord(helperColl, true)
	if err != nil {
		return err
	}
	if len(indexerRecord.Indexer) > 0 && indexerRecord.IsBuildDone {
		annServer.Index.Load(indexerRecord.Indexer)
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
		hashes, err := annServer.Index.GetHashes(cm.Vector(record.FeatureVec))
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

// popHashRecord drops record from collection by objectID (string Hex)
func (annServer *ANNServer) popHashRecord(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	database := annServer.MongoClient.GetDb(annServer.Config.DbName)
	helperRecord, err := db.GetHelperRecord(database.Collection(annServer.Config.HelperCollectionName), false)
	if err != nil {
		return err
	}
	hashesColl := database.Collection(helperRecord.HashCollName)
	err = db.DeleteRecords(hashesColl, bson.D{{"_id", objectID}})
	if err != nil {
		return err
	}
	return nil
}

// putHashRecord drops record from collection by objectID (string Hex)
func (annServer *ANNServer) putHashRecord(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	database := annServer.MongoClient.GetDb(annServer.Config.DbName)
	helperRecord, err := db.GetHelperRecord(database.Collection(annServer.Config.HelperCollectionName), false)
	if err != nil {
		return err
	}
	hashesColl := database.Collection(helperRecord.HashCollName)
	records, err := db.GetDbRecords(
		database.Collection(annServer.Config.DataCollectionName),
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
	err = db.SetRecords(hashesColl, recordInterfaces)
	if err != nil {
		return err
	}
	return nil
}

// getNeighbors returns filtered nearest neighbors sorted by distance in ascending order
func (annServer *ANNServer) getNeighbors(input RequestData) (*ResponseData, error) {
	inputVec := cm.Vector(input.Vec)
	var hashesRecords []db.HashesRecord
	database := annServer.MongoClient.GetDb(annServer.Config.DbName)
	if input.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(input.ID)
		if err != nil {
			return nil, err
		}
		helperRecord, err := db.GetHelperRecord(database.Collection(annServer.Config.HelperCollectionName), false)
		if err != nil {
			return nil, err
		}
		hashesColl := database.Collection(helperRecord.HashCollName)
		hashesRecords, err = db.GetHashesRecords(
			hashesColl,
			db.FindQuery{
				Limit: 1,
				Query: bson.D{{"_id", objectID}},
				Proj:  bson.M{"_id": 1},
			},
		)
		if err != nil {
			return nil, err
		}
	} else if !inputVec.IsZero() {
		hashes, err := annServer.Index.GetHashes(inputVec)
		helperRecord, err := db.GetHelperRecord(database.Collection(annServer.Config.HelperCollectionName), false)
		if err != nil {
			return nil, err
		}
		hashesColl := database.Collection(helperRecord.HashCollName)
		hashesQuery := bson.D{}
		for k, v := range hashes {
			hashesQuery = append(hashesQuery, bson.E{strconv.Itoa(k), v})
		}
		hashesRecords, err = db.GetHashesRecords(
			hashesColl,
			db.FindQuery{
				Limit: annServer.Config.MaxHashesNumber,
				Query: hashesQuery,
				Proj:  bson.M{"_id": 1},
			},
		)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("Get NN: object ID or vector must be provided")
	}

	vectorIDs := bson.A{}
	for idx := range hashesRecords {
		vectorIDs = append(vectorIDs, hashesRecords[idx].ID)
	}
	hashesRecords = nil

	vectorsCursor, err := db.GetCursor(
		database.Collection(annServer.Config.DataCollectionName),
		db.FindQuery{
			Query: bson.D{{"_id", bson.D{{"$in", vectorIDs}}}},
			Proj:  bson.M{"_id": 1, "featureVec": 1},
		},
	)
	if err != nil {
		return nil, err
	}

	neighbors := make([]ResponseRecord, annServer.Config.MaxNN)
	var idx int = 0
	var candidate db.VectorRecord
	for vectorsCursor.Next(context.Background()) && idx <= annServer.Config.MaxNN {
		if err := vectorsCursor.Decode(&candidate); err != nil {
			continue
		}
		hexID := candidate.ID.Hex()
		dist := annServer.Index.GetDist(inputVec, cm.Vector(candidate.FeatureVec))
		if dist <= annServer.Config.DistanceThrsh {
			neighbors[idx] = ResponseRecord{
				ID:   hexID,
				Dist: dist,
			}
			idx++
		}
	}
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].Dist < neighbors[j].Dist
	})
	return &ResponseData{Neighbors: neighbors}, nil
}
