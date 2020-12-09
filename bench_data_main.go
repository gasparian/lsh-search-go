package main

import (
	"log"
	"os"
	"strconv"

	"gonum.org/v1/hdf5"
	annb "vector-search-go/annbench"
	"vector-search-go/db"
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

	f, err := hdf5.OpenFile("./annbench/deep-image-96-angular.hdf5", hdf5.F_ACC_RDWR)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	{
		featuresTest, err := annb.GetVectorsFromHDF5(f, "test")
		if err != nil {
			log.Fatal(err)
		}
		neighbors, err := annb.GetNeighborsFromHDF5(f, "neighbors")
		if err != nil {
			log.Fatal(err)
		}

		err = annb.LoadDatasetMongoDb(vectorsTestCollection, featuresTest, neighbors, batchSize)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Test data has been saved to mongo!")
	}

	{
		featuresTrain, err := annb.GetVectorsFromHDF5(f, "train")
		if err != nil {
			log.Fatal(err)
		}
		err = annb.LoadDatasetMongoDb(vectorsTrainCollection, featuresTrain, []db.NeighborsIds{}, batchSize)
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
