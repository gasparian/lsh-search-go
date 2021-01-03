package main

import (
	"fmt"
	"os"
	"strconv"

	annb "lsh-search-service/annbench"
	cl "lsh-search-service/client"
	cm "lsh-search-service/common"
	"lsh-search-service/db"
)

var (
	dbLocation         = os.Getenv("MONGO_ADDR")
	dbName             = os.Getenv("DB_NAME")
	dbtimeOut, _       = strconv.Atoi(os.Getenv("DB_CLIENT_TIMEOUT"))
	batchSize, _       = strconv.Atoi(os.Getenv("BATCH_SIZE"))
	dataCollectionName = os.Getenv("DATA_COLLECTION_NAME")
	testCollectionName = os.Getenv("TEST_COLLECTION_NAME")
)

func main() {
	logger := cm.GetNewLogger()
	mongodb, err := db.New(
		db.Config{
			DbLocation: dbLocation,
			DbName:     dbName,
		},
	)
	if err != nil {
		logger.Err.Fatal(err)
	}

	benchClient := annb.BenchClient{
		Client: cl.New(cl.Config{
			ServerAddress: "http://192.168.0.132",
			Timeout:       dbtimeOut,
		}),
		Mongo:          mongodb,
		Logger:         logger,
		TestCollection: mongodb.GetCollection(testCollectionName),
	}
	defer benchClient.Mongo.Disconnect()

	hashCollSize, err := benchClient.Client.GetHashCollSize()
	if err != nil {
		logger.Err.Fatal(err)
	}
	datasetSize, err := benchClient.Mongo.GetCollSize(dataCollectionName)
	if err != nil {
		logger.Err.Fatal(err)
	}
	logger.Info.Printf("Index size: %v; Full dataset size: %v", hashCollSize, datasetSize)
	if hashCollSize != 0 && hashCollSize != datasetSize {
		logger.Err.Fatal(fmt.Errorf("Search index size not equals to the bench dataset size"))
	}
	if hashCollSize == 0 {
		err = benchClient.PopulateDataset(batchSize, dataCollectionName)
		if err != nil {
			logger.Err.Fatal(err)
		}
	}
	// thrshs := []float64{0.05, 0.1, 0.15, 0.2, 0.25, 0.3, 0.4, 0.5, 0.6, 0.7}
	thrshs := []float64{0.1} // DEBUG
	result, err := benchClient.Validate(thrshs)
	if err != nil {
		logger.Err.Fatal(err)
	}
	logger.Info.Println(result)
}
