package annbench

import (
	"go.mongodb.org/mongo-driver/bson"
	cl "lsh-search-service/client"
	cm "lsh-search-service/common"
	"lsh-search-service/db"
	"os"
	"sort"
	"time"
)

var (
	testCollectionName = os.Getenv("TEST_COLLECTION_NAME")
)

// BenchClient holds db for getting vectors from test collection
// and a client for performing requests to the running ann service
type BenchClient struct {
	Client cl.ANNClient
	Db     *db.MongoDatastore
	Logger *cm.Logger
}

// Recall returns ratio of relevant predictions over the all true relevant items
// both arrays MUST BE SORTED
func Recall(prediction, groundTruth []int32) float64 {
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
	var prediction []int32
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
			prediction = append(prediction, neighbor.OrigID)
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

// BuildIndex creates hasher object on server
// TO DO:
func (benchClient *BenchClient) BuildIndex() error {

	dataColl := annServer.Mongo.GetCollection(annServer.Config.Db.DataCollectionName)
	convMean, convStd, err := dataColl.GetAggregatedStats()
	if err != nil {
		return err
	}

	annServer.Logger.Info.Println(convMean) // DEBUG - check for not being [0]
	annServer.Logger.Info.Println(convStd)  // DEBUG - check for not being [0]

	// NOTE: fill the new collection with pointers to documents (_id) and fields with hashes
	cursor, err := dataColl.GetCursor(
		db.FindQuery{
			Limit: 0,
			Query: bson.D{},
		},
	)
	for cursor.Next(context.Background()) {
		hashesBatch, err := annServer.hashDbRecordsBatch(cursor, annServer.Config.App.BatchSize)
		if err != nil {
			return err
		}
		err = newHashColl.SetRecords(hashesBatch)
		if err != nil {
			return err
		}
	}

	dataColl := annServer.Mongo.GetCollection(annServer.Config.Db.DataCollectionName)
	records, err := dataColl.GetDbRecords(
		db.FindQuery{
			Limit: 1,
			Query: bson.D{{"_id", objectID}},
		},
	)
	if err != nil {
		return err
	}
}

// hashDbRecordsBatch accumulates db documents in a batch of desired length and calculates hashes
// TO DO: refactor to use on index building (look up ^)
func (annServer *ANNServer) hashDbRecordsBatch(cursor *mongo.Cursor, batchSize int) ([]interface{}, error) {
	batch := make([]interface{}, batchSize)
	batchID := 0
	for cursor.Next(context.Background()) {
		var record db.VectorRecord
		if err := cursor.Decode(&record); err != nil {
			continue
		}
		hashes, err := annServer.Hasher.GetHashes(cm.Vector(record.FeatureVec))
		if err != nil {
			return nil, err
		}
		batch[batchID] = db.HashesRecord{
			ID:     record.ID,
			Hashes: hashes,
		}
		batchID++
	}
	return batch[:batchID], nil
}
