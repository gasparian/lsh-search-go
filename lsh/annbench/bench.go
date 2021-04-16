package annbench

import (
	"context"
	cl "github.com/gasparian/similarity-search-go/lsh/client"
	cm "github.com/gasparian/similarity-search-go/lsh/common"
	"sort"
	"time"
)

// BenchClient holds storage for getting vectors from test collection
// and a client for performing requests to the running ann service
type BenchClient struct {
	Client cl.ANNClient
	Logger *cm.Logger
}

// Recall returns ratio of relevant predictions over the all true relevant items
// both arrays MUST BE SORTED
func Recall(prediction, groundTruth []uint64) float64 {
	valid := 0
	for i := range prediction {
		if prediction[i] == groundTruth[i] {
			valid++
		}
	}
	return float64(valid) / float64(len(groundTruth))
}

// ValidateThrsh takes the distance threshold and returns recall value
func (benchClient *BenchClient) ValidateThrsh(results []storage.VectorRecord, thrsh float64) (float64, error) {
	var averageRecall float64 = 0.0
	var prediction []uint64
	for _, result := range results {
		sort.Slice(result.NeighborsIds, func(i, j int) bool {
			return result.NeighborsIds[i] < result.NeighborsIds[j]
		})
		neighborsIDs, err := benchClient.Client.GetNeighbors(result.FeatureVec)
		if err != nil {
			return 0.0, err
		}
		prediction = nil
		for _, neighborID := range neighborsIDs {
			prediction = append(prediction, neighborID)
		}
		averageRecall += Recall(prediction, result.NeighborsIds)
	}
	return averageRecall / float64(len(results)), nil
}

// Validate takes the array of distance thresholds and returns array of recall values
func (benchClient *BenchClient) Validate(thrshs []float64) ([]float64, error) {
	metrics := make([]float64, len(thrshs))
	results, err := db.GetDbRecords(benchClient.TestCollection, db.FindQuery{Proj: bson.M{"featureVec": 1}})
	if err != nil {
		return nil, err
	}
	for _, thrsh := range thrshs {
		start := time.Now()
		recall, err := benchClient.ValidateThrsh(results, thrsh)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, recall)
		elapsed := time.Since(start)
		benchClient.Logger.Info.Printf("Elapsed time: %v; Thrsh: %v; Recall: %v", elapsed, thrsh, recall)
	}
	return metrics, nil
}

// Populate put vectors into search index
func (benchClient *BenchClient) PopulateDataset(batchSize int, dataCollName string) error {
	dataColl := benchClient.Mongo.GetCollection(dataCollName)
	convMean, convStd, err := db.GetAggregatedStats(dataColl)
	if err != nil {
		return err
	}

	benchClient.Logger.Info.Println(convMean) // DEBUG - check for not being [0]
	benchClient.Logger.Info.Println(convStd)  // DEBUG - check for not being [0]

	benchClient.Client.BuildHasher(convMean, convStd)

	cursor, err := dataColl.GetCursor(db.FindQuery{})
	for cursor.Next(context.Background()) {
		err = benchClient.putBatch(cursor, batchSize)
		if err != nil {
			return err
		}
	}
	return nil
}

// putBatch accumulates db documents in a batch of desired length and calculates hashes
func (benchClient *BenchClient) putBatch(cursor *mongo.Cursor, batchSize int) error {
	batch := make([]cm.RequestData, batchSize)
	batchID := 0
	for cursor.Next(context.Background()) {
		var record db.VectorRecord
		if err := cursor.Decode(&record); err != nil {
			continue
		}
		batch[batchID] = cm.RequestData{
			SecondaryID: record.SecondaryID,
			Vec:         record.FeatureVec,
		}
		batchID++
	}
	err := benchClient.Client.PutHashes(batch[:batchID])
	if err != nil {
		return err
	}
	return nil
}

// var (
// 	dbLocation          = os.Getenv("MONGO_ADDR")
// 	batchSize, _        = strconv.Atoi(os.Getenv("BATCH_SIZE"))
// 	trainCollectionName = os.Getenv("COLLECTION_NAME")
// 	testCollectionName  = os.Getenv("TEST_COLLECTION_NAME")
// )

// func main() {
// 	logger := cm.GetNewLogger()
// 	config := storage.Config{
// 		DbLocation: dbLocation,
// 	}
// 	logger.Info.Println("Db communication setup")
// 	mongodb, err := storage.New(config)
// 	if err != nil {
// 		logger.Err.Fatal(err)
// 	}
// 	defer mongodb.Disconnect()

// 	logger.Info.Println("Creating train collection...")
// 	mongodb.DropCollection(trainCollectionName)
// 	vectorsTrainCollection, err := mongodb.CreateCollection(trainCollectionName)
// 	if err != nil {
// 		logger.Err.Fatal(err)
// 	}
// 	logger.Info.Println("Creating test collection...")
// 	mongodb.DropCollection(testCollectionName)
// 	vectorsTestCollection, err := mongodb.CreateCollection(testCollectionName)
// 	if err != nil {
// 		logger.Err.Fatal(err)
// 	}

// 	logger.Info.Println("Opening the hdf5 bench dataset...")
// 	f, err := hdf5.OpenFile("./annbench/deep-image-96-angular.hdf5", hdf5.F_ACC_RDWR)
// 	if err != nil {
// 		logger.Err.Fatal(err)
// 	}
// 	defer f.Close()

// 	{
// 		logger.Info.Println("Creating test dataset...")
// 		featuresTest, err := annb.GetVectorsFromHDF5(f, "test")
// 		if err != nil {
// 			logger.Err.Fatal(err)
// 		}
// 		neighbors, err := annb.GetNeighborsFromHDF5(f, "neighbors")
// 		if err != nil {
// 			logger.Err.Fatal(err)
// 		}

// 		err = annb.UploadDatasetMongoDb(vectorsTestCollection, featuresTest, neighbors, batchSize)
// 		if err != nil {
// 			logger.Err.Fatal(err)
// 		}
// 		logger.Info.Println("Test data has been saved to mongo!")
// 	}

// 	{
// 		logger.Info.Println("Creating train dataset...")
// 		featuresTrain, err := annb.GetVectorsFromHDF5(f, "train")
// 		if err != nil {
// 			logger.Err.Fatal(err)
// 		}
// 		err = annb.UploadDatasetMongoDb(vectorsTrainCollection, featuresTrain, []storage.NeighborsIds{}, batchSize)
// 		if err != nil {
// 			logger.Err.Fatal(err)
// 		}
// 		logger.Info.Println("Train data has been saved to mongo!")
// 	}

// 	// DEBUG Index
// 	// vectorsTestCollection := mongodb.GetCollection(testCollectionName)
// 	// vectorsTrainCollection := mongodb.GetCollection(trainCollectionName)

// 	logger.Info.Println("Creating index on secondary id field...")
// 	err = vectorsTestCollection.CreateIndexesByFields([]string{"secondaryId"}, true)
// 	if err != nil {
// 		logger.Err.Fatal(err)
// 	}
// 	err = vectorsTrainCollection.CreateIndexesByFields([]string{"secondaryId"}, true)
// 	if err != nil {
// 		logger.Err.Fatal(err)
// 	}
// 	logger.Info.Println("Index has been created!")
// }
