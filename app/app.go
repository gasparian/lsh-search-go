package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	alg "vector-search-go/lsh"
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

	return searchHandler, nil
}

// BuildIndexerHandler updates the existing db documents with the
// new computed hashes based on dataset stats;
// Also we need to store somewhere the build status
// to prevent any db requests during this process
// TO DO
func (annServer *ANNServer) BuildIndexerHandler() error {

	/*
		TO DO: add here retrieving of the LSHIndex object from the database
		       or create new one if couldn't find it;
		       also it's better to set the special key to know the status
		       of build, to prevent other "workers" to do any work
	*/

	database := annServer.MongoClient.GetDb(dbName)
	coll := database.Collection(dataCollectionName)

	convMean, convStd, err := db.GetAggregatedStats(coll)
	if err != nil {
		return err
	}

	log.Println(convMean.Values) // DEBUG
	log.Println(convStd.Values)  // DEBUG

	lshIndex, err := alg.NewLSHIndex(convMean, convStd)
	if err != nil {
		return err
	}

	annServer.Index = lshIndex

	log.Println(annServer.Index.Entries[0]) // DEBUG

	return nil
}

// UpdateDbHashes updates entries in the data collection with the new set of hashes
// TO DO
func (annServer *ANNServer) UpdateDbHashes() {

}

// DbLoadIndexer loads indexer object from db if it exists
// TO DO
func (annServer *ANNServer) DbLoadIndexer() error {
	return nil
}

// DbSaveIndexer uploads indexer object to the db
// TO DO
func (annServer *ANNServer) DbSaveIndexer() error {
	return nil
}

// DbLockIndexer updates status in special document inside the db
// so other service workers blocks any operation while search index is updated
// TO DO
func (annServer *ANNServer) DbLockIndexer() error {
	return nil
}

// DbUnlockIndexer updates status in special document inside the db
// so other service workers could start using created search index
// and retrieve fresh indexer object
// TO DO
func (annServer *ANNServer) DbUnlockIndexer() error {
	return nil
}

// PutHashHandler calculates and updates the document with hashes
// TO DO
func (annServer *ANNServer) PutHashHandler(w http.ResponseWriter, r *http.Request) {

}

// PopHashHandler drops fields with hashes from the queried db entry
// TO DO
func (annServer *ANNServer) PopHashHandler(w http.ResponseWriter, r *http.Request) {

}

// GetNeighborsHandler makes query to the db and returns all
// Neighbors in the MaxDist
// TO DO
func (annServer *ANNServer) GetNeighborsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// DEBUG CODE
	database := annServer.MongoClient.GetDb(dbName)
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
