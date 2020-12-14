package app

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	cm "vector-search-go/common"
	"vector-search-go/db"
	alg "vector-search-go/lsh"
)

var (
	dbLocation           = os.Getenv("MONGO_ADDR")
	dbName               = os.Getenv("DB_NAME")
	dataCollectionName   = os.Getenv("COLLECTION_NAME")
	testCollectionName   = os.Getenv("TEST_COLLECTION_NAME")
	helperCollectionName = os.Getenv("HELPER_COLLECTION_NAME")
	batchSize, _         = strconv.Atoi(os.Getenv("BATCH_SIZE"))
	maxHashesNo          = int(10e5)
	maxNN, _             = strconv.Atoi(os.Getenv("MAX_NN"))
	distanceThrsh, _     = strconv.ParseFloat(os.Getenv("DISTANCE_THRSH"), 32)
)

// HealthCheck just checks that server is up and running;
// also gives back list of available methods
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var raw map[string]interface{}
	err := json.Unmarshal(HelloMessage, &raw)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	out, _ := json.Marshal(raw)
	w.Write(out)
}

// NewANNServer returns empty index object with initialized mongo client
func NewANNServer() (ANNServer, error) {
	mongodb, err := db.GetDbClient(dbLocation)
	if err != nil {
		log.Println("Creating db client: " + err.Error())
		return ANNServer{}, err
	}
	// defer mongodb.Disconnect() // should be placed on the upper level

	searchHandler := ANNServer{
		MongoClient: *mongodb,
	}
	err = searchHandler.LoadIndexer()
	if err != nil {
		return ANNServer{}, err
	}
	return searchHandler, nil
}

// LoadIndexer load indexer from the db if it exists
func (annServer *ANNServer) LoadIndexer() error {
	database := annServer.MongoClient.GetDb(dbName)
	helperColl := database.Collection(helperCollectionName)
	result, err := db.GetHelperRecord(helperColl)
	if err != nil {
		return err
	}
	if len(result.Indexer) > 0 && result.IsBuildDone {
		annServer.Index.Load(result.Indexer)
		return nil
	}
	return errors.New("Can't load indexer object")
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
		hashes, err := annServer.Index.GetHashes(
			cm.Vector{
				Values: record.FeatureVec,
				Size:   len(record.FeatureVec),
			},
		)
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

// BuildIndexerHandler updates the existing db documents with the
// new computed hashes based on dataset stats;
// TO DO:
//     after the indexer object is ready - we must call every other worker to load fresh model
//     make some keys verification, so not every user can spam a build operation
//     make it in async way (in a goroutine)
func (annServer *ANNServer) BuildIndexerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	database := annServer.MongoClient.GetDb(dbName)
	coll := database.Collection(dataCollectionName)
	db.CreateCollection(database, helperCollectionName)
	helperColl := database.Collection(helperCollectionName)

	// TO DO: check if the previous build has been done
	// Start build process
	err := db.UpdateField(
		helperColl,
		bson.D{
			{"indexer", bson.D{
				{"$exists", true},
			}}},
		bson.D{
			{"$set", bson.D{
				{"isBuildDone", false}},
			}})

	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	convMean, convStd, err := db.GetAggregatedStats(coll)
	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println(convMean.Values) // DEBUG - check for not being [0]
	log.Println(convStd.Values)  // DEBUG - check for not being [0]

	lshIndex, err := alg.NewLSHIndex(convMean, convStd)
	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	annServer.Index = lshIndex

	log.Println(annServer.Index.Entries[0]) // DEBUG - check for not being [0]

	lshSerialized, err := lshIndex.Dump()
	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Getting old hash collection name
	oldHelperRecord, err := db.GetHelperRecord(helperColl)
	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Generating and saving new hash collection name
	newHashCollName, err := cm.GetRandomID()
	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = db.UpdateField(
		helperColl,
		bson.D{
			{"indexer", bson.D{
				{"$exists", true},
			}}},
		bson.D{
			{"$set", bson.D{
				{"indexer", lshSerialized},
				{"hashCollName", newHashCollName},
			}}})

	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// create new collection for storing the newly generated hashes, while keeping the old one
	err = db.CreateCollection(database, newHashCollName)
	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// fill the new collection with pointers to documents (_id) and fields with hashes
	newHashColl := database.Collection(newHashCollName)
	cursor, err := db.GetCursor(
		coll,
		db.FindQuery{
			Limit: 0,
			Query: bson.D{},
		},
	)
	for cursor.Next(context.Background()) {
		hashesBatch, err := annServer.hashDbRecordsBatch(cursor, batchSize)
		if err != nil {
			log.Println("Building index: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = db.SetRecords(newHashColl, hashesBatch)
		if err != nil {
			log.Println("Building index: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	// create indexes for the all new fields
	hashesColl := database.Collection(newHashCollName)
	err = db.CreateIndexesByFields(hashesColl, annServer.Index.HashFieldsNames, false)
	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// drop old collection with hashes
	if oldHelperRecord.HashCollName != "" {
		err = db.DropCollection(database, oldHelperRecord.HashCollName)
		if err != nil {
			log.Println("Building index: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	err = db.UpdateField(
		helperColl,
		bson.D{
			{"indexer", bson.D{
				{"$exists", true},
			}}},
		bson.D{
			{"$set", bson.D{
				{"isBuildDone", true}},
			}})

	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// popHashRecord drops record from collection by objectID (string Hex)
func (annServer *ANNServer) popHashRecord(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	database := annServer.MongoClient.GetDb(dbName)
	helperRecord, err := db.GetHelperRecord(database.Collection(helperCollectionName))
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

// PopHashRecordHandler drops vector from the search index
// curl -v http://localhost:8080/check?id=kd8f9wfhsdfs9df
func (annServer *ANNServer) PopHashRecordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		params := r.URL.Query()
		// NOTE: id generated from mongodb ObjectID with Hex() method
		id, ok := params["id"]
		if !ok || len(id) == 0 {
			log.Println("Pop hash record: object id must be specified")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if id[0] == "" {
			log.Println("Pop hash record: object id must be specified")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err := annServer.popHashRecord(id[0])
		if err != nil {
			log.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var input RequestData
		err = json.Unmarshal(body, &input)
		if err != nil {
			log.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = annServer.popHashRecord(input.ID)
		if err != nil {
			log.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

// putHashRecord drops record from collection by objectID (string Hex)
func (annServer *ANNServer) putHashRecord(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	database := annServer.MongoClient.GetDb(dbName)
	helperRecord, err := db.GetHelperRecord(database.Collection(helperCollectionName))
	if err != nil {
		return err
	}
	hashesColl := database.Collection(helperRecord.HashCollName)
	records, err := db.GetDbRecords(
		database.Collection(dataCollectionName),
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

// PutHashRecordHandler puts new vector to the search index (also updates the initial db??)
// curl -v -X POST -H "Content-Type: application/json" -d '{"id": "sdf87sdfsdf9s8dfb", "vec": []}' http://localhost:8080/put
func (annServer *ANNServer) PutHashRecordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		params := r.URL.Query()
		// NOTE: id generated from mongodb ObjectID with Hex() method
		id, ok := params["id"]
		if !ok || len(id) == 0 {
			log.Println("Pop hash record: object id must be specified")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if id[0] == "" {
			log.Println("Pop hash record: object id must be specified")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err := annServer.putHashRecord(id[0])
		if err != nil {
			log.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var input RequestData
		err = json.Unmarshal(body, &input)
		if err != nil {
			log.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = annServer.putHashRecord(input.ID)
		if err != nil {
			log.Println("Pop hash record: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

// getNeighbors returns filtered nearest neighbors sorted by distance in ascending order
func (annServer *ANNServer) getNeighbors(input RequestData) (*ResponseData, error) {
	inputVec := cm.NewVector(input.Vec)
	var hashesRecords []db.HashesRecord
	database := annServer.MongoClient.GetDb(dbName)
	if input.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(input.ID)
		if err != nil {
			return nil, err
		}
		helperRecord, err := db.GetHelperRecord(database.Collection(helperCollectionName))
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
		helperRecord, err := db.GetHelperRecord(database.Collection(helperCollectionName))
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
				Limit: maxHashesNo,
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
		database.Collection(dataCollectionName),
		db.FindQuery{
			Query: bson.D{{"_id", bson.D{{"$in", vectorIDs}}}},
			Proj:  bson.M{"_id": 1, "featureVec": 1},
		},
	)
	if err != nil {
		return nil, err
	}

	neighbors := make([]ResponseRecord, maxNN)
	var idx int = 0
	var candidate db.VectorRecord
	for vectorsCursor.Next(context.Background()) && idx <= maxNN {
		if err := vectorsCursor.Decode(&candidate); err != nil {
			continue
		}
		hexID := candidate.ID.Hex()
		dist := annServer.Index.GetDist(inputVec, cm.NewVector(candidate.FeatureVec))
		if dist <= distanceThrsh {
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
	return &ResponseData{NN: neighbors}, nil
}

// GetNeighborsHandler makes query to the db and returns all neighbors in the MaxDist
func (annServer *ANNServer) GetNeighborsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Get NN: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var input RequestData
		err = json.Unmarshal(body, &input)
		if err != nil {
			log.Println("Get NN: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		result, err := annServer.getNeighbors(input)
		if err != nil {
			log.Println("Get NN: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		jsonResp, err := json.Marshal(result)
		if err != nil {
			log.Println("Get NN: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(jsonResp)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}
