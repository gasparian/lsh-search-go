package main

import (
	"log"
	"os"
	"strconv"

	"vector-search-go/db"

	"gonum.org/v1/hdf5"
)

var (
	dbLocation          = os.Getenv("MONGO_ADDR")
	dbName              = os.Getenv("DB_NAME")
	batchSize, _        = strconv.Atoi(os.Getenv("BATCH_SIZE"))
	trainCollectionName = os.Getenv("COLLECTION_NAME")
	testCollectionName  = os.Getenv("TEST_COLLECTION_NAME")
)

func main() {
	mongodb, err := db.GetDbClient(dbLocation)
	if err != nil {
		log.Fatal(err)
	}
	defer mongodb.Disconnect()

	database := mongodb.GetDb(dbName)
	vectorsTrainCollection := database.Collection(trainCollectionName)
	vectorsTestCollection := database.Collection(testCollectionName)

	f, err := hdf5.OpenFile("./db/deep-image-96-angular.hdf5", hdf5.F_ACC_RDWR)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	{
		featuresTest, err := db.GetVectorsFromHDF5(f, "test")
		if err != nil {
			log.Fatal(err)
		}
		neighbors, err := db.GetNeighborsFromHDF5(f, "neighbors")
		if err != nil {
			log.Fatal(err)
		}

		err = db.LoadDatasetMongoDb(vectorsTestCollection, featuresTest, neighbors, batchSize)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Test data has been saved to mongo!")
	}

	{
		featuresTrain, err := db.GetVectorsFromHDF5(f, "train")
		if err != nil {
			log.Fatal(err)
		}
		err = db.LoadDatasetMongoDb(vectorsTrainCollection, featuresTrain, []db.NeighborsIds{}, batchSize)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Train data has been saved to mongo!")
	}

	log.Println("Creating index on OrigId field...")
	err = mongodb.CreateIndexesByFields(vectorsTestCollection, []string{"OrigId"}, true)
	if err != nil {
		log.Fatal(err)
	}
	err = mongodb.CreateIndexesByFields(vectorsTrainCollection, []string{"OrigId"}, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Index has been created!")
}
