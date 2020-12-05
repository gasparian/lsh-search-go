package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	alg "vector-search-go/algorithm"
	"vector-search-go/db"
)

var (
	dbLocation         = os.Getenv("MONGO_ADDR")
	dbName             = os.Getenv("DB_NAME")
	dataCollectionName = os.Getenv("COLLECTION_NAME")
	testCollectionName = os.Getenv("TEST_COLLECTION_NAME")
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

	convMean, convStd, err := db.GetAggregatedStats(coll)
	if err != nil {
		return SearchIndexHandler{}, err
	}

	log.Println(convMean.Values) // DEBUG
	log.Println(convStd.Values)  // DEBUG

	// TO DO: If there is no existing Index object stored in the database
	lshIndex, err := alg.NewLSHIndex(convMean, convStd)
	if err != nil {
		return SearchIndexHandler{}, err
	}

	searchHandler := SearchIndexHandler{
		Index:       lshIndex,
		MongoClient: *mongodb,
	}

	log.Println(searchHandler.Index.Entries[0]) // DEBUG

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
