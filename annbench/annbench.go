package annbench

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	cl "lsh-search-service/client"
	cm "lsh-search-service/common"
	"lsh-search-service/db"
	"os"
	"sort"
	"time"
)

var (
	testCollectionName = os.Getenv("TEST_COLLECTION_NAME")
	dataCollectionName = os.Getenv("DATA_COLLECTION_NAME")
)

// BenchClient holds db for getting vectors from test collection
// and a client for performing requests to the running ann service
type BenchClient struct {
	Client               cl.ANNClient
	Db                   *db.MongoDatastore
	Logger               *cm.Logger
	HelperCollectionName string
}

// Recall returns ratio of relevant predictions over the all true relevant items
// both arrays MUST BE SORTED
func Recall(prediction, groundTruth []int) float64 {
	valid := 0
	for i := range prediction {
		if prediction[i] == groundTruth[i] {
			valid++
		}
	}
	return float64(valid) / float64(len(groundTruth))
}

// ValidateThrsh takes the distance threshold and returns recall value
func (benchClient *BenchClient) ValidateThrsh(results []db.VectorRecord, thrsh float64) (float64, error) {
	var averageRecall float64 = 0.0
	var prediction []int
	for _, result := range results {
		sort.Slice(result.NeighborsIds, func(i, j int) bool {
			return result.NeighborsIds[i] < result.NeighborsIds[j]
		})
		respData, err := benchClient.Client.GetNeighbors(result.FeatureVec)
		if err != nil {
			return 0.0, err
		}
		prediction = nil
		for _, neighbor := range respData.Neighbors {
			prediction = append(prediction, neighbor.SecondaryID)
		}
		averageRecall += Recall(prediction, result.NeighborsIds)
	}
	return averageRecall / float64(len(results)), nil
}

// Validate takes the array of distance thresholds and returns array of recall values
func (benchClient *BenchClient) Validate(thrshs []float64) ([]float64, error) {
	metrics := make([]float64, len(thrshs))
	testColl := benchClient.Db.GetCollection(testCollectionName)
	results, err := testColl.GetDbRecords(db.FindQuery{Proj: bson.M{"featureVec": 1}})
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
func (benchClient *BenchClient) Populate(batchSize int) error {
	dataColl := benchClient.Db.GetCollection(dataCollectionName)
	convMean, convStd, err := dataColl.GetAggregatedStats()
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
