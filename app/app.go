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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	alg "vector-search-go/algorithm"
	"vector-search-go/db"
)

var (
	dbLocation         = os.Getenv("MONGO_ADDR")
	dbName             = os.Getenv("DB_NAME")
	dataCollectionName = os.Getenv("COLLECTION_NAME")
	testCollectionName = os.Getenv("TEST_COLLECTION_NAME")
	maxNPlanes, _      = strconv.Atoi(os.Getenv("MAX_N_PLANES"))
	nPermutes, _       = strconv.Atoi(os.Getenv("N_PERMUTS"))
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

func convertAggResult(inp interface{}) (alg.Vector, error) {
	val, ok := inp.(primitive.A)
	if !ok {
		return alg.Vector{}, errors.New("Type conversion failed")
	}
	conv := alg.Vector{
		Values: make([]float64, len(val)),
		Size:   len(val),
	}
	for i := range conv.Values {
		v, ok := val[i].(float64)
		if !ok {
			return alg.Vector{}, errors.New("Type conversion failed")
		}
		conv.Values[i] = v
	}
	return conv, nil
}

// BuildIndex updates the existing db documents with the
// new computed hashes based on dataset stats;
// Also we need to store somewhere the build status
// to prevent any db requests during this process
func BuildIndex() (SearchIndexHandler, error) {
	mongodb, err := db.GetDbClient(dbLocation)
	if err != nil {
		log.Println("Creating db client: " + err.Error())
		return SearchIndexHandler{}, err
	}
	// defer mongodb.Disconnect() // should be placed on some upper level

	/*
		TO DO: add here retrieving of the LSHIndex object from the database
		       or create new one if couldn't find it;
		       also it's better to set the special key to know the status
		       of build, to prevent other "workers" to do any work
	*/

	database := mongodb.GetDb(dbName)
	coll := database.Collection(dataCollectionName)

	results, err := db.GetAggregation(coll, db.GroupMeanStd)
	if err != nil {
		log.Println("Making db aggregation: " + err.Error())
		return SearchIndexHandler{}, err
	}
	convMean, err := convertAggResult(results[0]["avg"])
	if err != nil {
		log.Println("Parsing aggregation result: " + err.Error())
		return SearchIndexHandler{}, err
	}
	convStd, err := convertAggResult(results[0]["std"])
	if err != nil {
		log.Println("Parsing aggregation result: " + err.Error())
		return SearchIndexHandler{}, err
	}

	log.Println(convMean.Values) // DEBUG
	log.Println(convStd.Values)  // DEBUG

	searchHandler := SearchIndexHandler{
		Index: alg.LSHIndex{
			Entries: make([]alg.LSHIndexRecord, nPermutes),
		},
		MongoClient: *mongodb,
	}

	var tmpLSHIndex alg.LSHIndexRecord
	for i := 0; i < nPermutes; i++ {
		tmpLSHIndex, err = alg.NewLSHIndexRecord(convMean, convStd, maxNPlanes)
		if err != nil {
			return SearchIndexHandler{}, err
		}
		searchHandler.Index.Entries[i] = tmpLSHIndex
	}

	log.Println(searchHandler.Index.Entries[0]) // DEBUG

	// If the new indexer object has been created -
	// - update all documents in the db

	return searchHandler, nil
}

// GetNeighbors makes query to the db and returns all
// Neighbors in the MaxDist
func (searchIndex *SearchIndexHandler) GetNeighbors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// DEBUG CODE
	database := searchIndex.MongoClient.GetDb(dbName)
	coll := database.Collection(dataCollectionName)

	opts := options.Find().SetLimit(2)
	// should be searching for all permutes at hashes field
	cursor, err := coll.Find(context.TODO(), bson.D{{"origId", bson.M{"$in": []int{1, 3}}}}, opts)
	if err != nil {
		log.Println("Get method find: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Println("Get method cursor: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	for _, result := range results {
		log.Println(result)
	}
	w.WriteHeader(http.StatusOK)
	// DEBUG CODE
}

// PutToIndex updates the document with
// calculated search hashes
func (searchIndex *SearchIndexHandler) PutToIndex(w http.ResponseWriter, r *http.Request) {

}

// PopFromIndex drops fields with hashes from
// the queried document
func (searchIndex *SearchIndexHandler) PopFromIndex(w http.ResponseWriter, r *http.Request) {

}
