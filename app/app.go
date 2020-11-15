package app

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"vector-search-go/db"
)

var (
	dbLocation         = os.Getenv("MONGO_ADDR")
	dbName             = os.Getenv("DB_NAME")
	collectionName     = os.Getenv("COLLECTION_NAME")
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

func convertAggResult(inp interface{}) ([]float64, error) {
	val, ok := inp.(primitive.A)
	if !ok {
		return nil, errors.New("Type conversion failed")
	}
	conv := make([]float64, len(val))
	for i := range conv {
		v, ok := val[i].(float64)
		if !ok {
			return nil, errors.New("Type conversion failed")
		}
		conv[i] = v
	}
	return conv, nil
}

// BuildIndex updates the existing db documents with the
// new computed hashes based on dataset stats and
// config parameters
func BuildIndex(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Content-Type", "application/json")
	mongodb, err := db.GetDbClient(dbLocation)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Creating db client: " + err.Error())
	}
	defer mongodb.Disconnect()
	database := mongodb.GetDb(dbName)
	coll := database.Collection(collectionName)

	results, err := db.GetAggregation(coll, db.GroupMeanStd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Making db aggregation: " + err.Error())
		return
	}
	convMean, err := convertAggResult(results[0]["avg"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Parsing aggregation result: " + err.Error())
		return
	}
	convStd, err := convertAggResult(results[0]["std"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Parsing aggregation result: " + err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Println(convMean)
	log.Println(convStd)
}
