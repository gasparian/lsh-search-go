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
	config := db.Config{
		DbLocation: dbLocation,
		DbName:     dbName,
	}
	mongodb, err := db.GetDbClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer mongodb.Disconnect()

	vectorsTrainCollection := mongodb.GetCollection(trainCollectionName)
	vectorsTestCollection := mongodb.GetCollection(testCollectionName)

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

		err = annb.UploadDatasetMongoDb(vectorsTestCollection, featuresTest, neighbors, batchSize)
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
		err = annb.UploadDatasetMongoDb(vectorsTrainCollection, featuresTrain, []db.NeighborsIds{}, batchSize)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Train data has been saved to mongo!")
	}

	log.Println("Creating index on OrigId field...")
	err = vectorsTestCollection.CreateIndexesByFields([]string{"OrigId"}, true)
	if err != nil {
		log.Fatal(err)
	}
	err = vectorsTrainCollection.CreateIndexesByFields([]string{"OrigId"}, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Index has been created!")
}
