package main

import (
	"os"
	"strconv"

	"gonum.org/v1/hdf5"
	annb "vector-search-go/annbench"
	cm "vector-search-go/common"
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
	logger := cm.GetNewLogger()
	config := db.Config{
		DbLocation: dbLocation,
		DbName:     dbName,
	}
	logger.Info.Println("Db communication setup")
	mongodb, err := db.GetDbClient(config)
	if err != nil {
		logger.Err.Fatal(err)
	}
	defer mongodb.Disconnect()

	logger.Info.Println("Creating train collection...")
	mongodb.DropCollection(trainCollectionName)
	vectorsTrainCollection, err := mongodb.CreateCollection(trainCollectionName)
	if err != nil {
		logger.Err.Fatal(err)
	}
	logger.Info.Println("Creating test collection...")
	mongodb.DropCollection(testCollectionName)
	vectorsTestCollection, err := mongodb.CreateCollection(testCollectionName)
	if err != nil {
		logger.Err.Fatal(err)
	}

	logger.Info.Println("Opening the hdf5 bench dataset...")
	f, err := hdf5.OpenFile("./annbench/deep-image-96-angular.hdf5", hdf5.F_ACC_RDWR)
	if err != nil {
		logger.Err.Fatal(err)
	}
	defer f.Close()

	{
		logger.Info.Println("Creating test dataset...")
		featuresTest, err := annb.GetVectorsFromHDF5(f, "test")
		if err != nil {
			logger.Err.Fatal(err)
		}
		neighbors, err := annb.GetNeighborsFromHDF5(f, "neighbors")
		if err != nil {
			logger.Err.Fatal(err)
		}

		err = annb.UploadDatasetMongoDb(vectorsTestCollection, featuresTest, neighbors, batchSize)
		if err != nil {
			logger.Err.Fatal(err)
		}
		logger.Info.Println("Test data has been saved to mongo!")
	}

	{
		logger.Info.Println("Creating train dataset...")
		featuresTrain, err := annb.GetVectorsFromHDF5(f, "train")
		if err != nil {
			logger.Err.Fatal(err)
		}
		err = annb.UploadDatasetMongoDb(vectorsTrainCollection, featuresTrain, []db.NeighborsIds{}, batchSize)
		if err != nil {
			logger.Err.Fatal(err)
		}
		logger.Info.Println("Train data has been saved to mongo!")
	}

	logger.Info.Println("Creating index on OrigId field...")
	err = vectorsTestCollection.CreateIndexesByFields([]string{"OrigId"}, true)
	if err != nil {
		logger.Err.Fatal(err)
	}
	err = vectorsTrainCollection.CreateIndexesByFields([]string{"OrigId"}, true)
	if err != nil {
		logger.Err.Fatal(err)
	}
	logger.Info.Println("Index has been created!")
}
