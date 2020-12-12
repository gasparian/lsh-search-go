package app

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	// "go.mongodb.org/mongo-driver/mongo/options"
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
	if len(result.Indexer) > 0 && result.Available {
		annServer.Index.Load(result.Indexer)
		return nil
	}
	return errors.New("Can't load indexer object")
}

// hashDbRecordsBatch accumulates db documents in a batch of desired length
func (annServer *ANNServer) hashDbRecordsBatch(cursor *mongo.Cursor, batchSize int) ([]interface{}, error) {
	batch := make([]interface{}, batchSize)
	batchID := 0
	for cursor.Next(context.Background()) {
		var record db.VectorRecord
		if err := cursor.Decode(&record); err != nil {
			continue
		}
		hashes, err := annServer.Index.GetHashes(cm.Vector{
			Values: record.FeatureVec,
			Size:   len(record.FeatureVec),
		})
		if err != nil {
			return nil, err
		}
		batch[batchID] = db.HashesRecord{
			OrigDocumentID: record.ID,
			Hashes:         hashes,
		}
		batchID++
	}
	return batch[:batchID], nil
}

// BuildIndexerHandler updates the existing db documents with the
// new computed hashes based on dataset stats;
// TO DO:
//     make it in async way (create goroutine??)
//     make some keys verification, so not every user can spam a build operation
func (annServer *ANNServer) BuildIndexerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	database := annServer.MongoClient.GetDb(dbName)
	coll := database.Collection(dataCollectionName)

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

	db.CreateCollection(database, helperCollectionName)
	helperColl := database.Collection(helperCollectionName)

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
				{"available", false},
				{"hashCollName", newHashCollName},
			}}})

	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// create new collection for storing the newly generated hashes, while keeping the old one (db.CreateCollection)
	err = db.CreateCollection(database, newHashCollName)
	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// fill the new collection with pointers to documents (_id) and fields with hashes
	newHashColl := database.Collection(newHashCollName)
	cursor, err := db.GetCursor(coll, -1, bson.D{})
	for cursor.Next(context.TODO()) {
		hashesBatch, err := annServer.hashDbRecordsBatch(cursor, batchSize)
		if err != nil {
			log.Println("Building index: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = db.SetData(newHashColl, hashesBatch)
		if err != nil {
			log.Println("Building index: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	// create indexes for the all new fields (db.CreateIndexesByFields)
	hashesColl := database.Collection(newHashCollName)
	err = db.CreateIndexesByFields(hashesColl, []string{}, false)
	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// drop old collection with hashes (db.DropCollection)
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
				{"available", true}},
			}})

	if err != nil {
		log.Println("Building index: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// PopHandler drops vector from the search index
// TO DO
func (annServer *ANNServer) PopHandler(w http.ResponseWriter, e *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// PutHandler puts new vector to the search index (also updates the initial db??)
// TO DO
func (annServer *ANNServer) PutHandler(w http.ResponseWriter, e *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// GetNeighborsHandler makes query to the db and returns all
// Neighbors in the MaxDist
// TO DO
func (annServer *ANNServer) GetNeighborsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// TO DO: check here that indexer object is available, and load if true (??)

	// DEBUG CODE
	database := annServer.MongoClient.GetDb(dbName)
	coll := database.Collection(dataCollectionName)

	results, err := db.GetDbRecords(coll, 2, bson.D{{"origId", bson.M{"$in": []int{1, 3}}}})
	if err != nil {
		log.Println("Find: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	for _, result := range results {
		log.Println(result)
	}
	w.WriteHeader(http.StatusOK)
	// DEBUG CODE
}
