package annbench

import (
	_ "context"
	// vc "github.com/gasparian/lsh-search-go/vector"
	_ "sort"
	_ "time"
)

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

// // ValidateThrsh takes the distance threshold and returns recall value
// func (benchClient *BenchClient) ValidateThrsh(results []storage.VectorRecord, thrsh float64) (float64, error) {
// 	var averageRecall float64 = 0.0
// 	var prediction []uint64
// 	for _, result := range results {
// 		sort.Slice(result.NeighborsIds, func(i, j int) bool {
// 			return result.NeighborsIds[i] < result.NeighborsIds[j]
// 		})
// 		neighborsIDs, err := benchClient.Client.GetNeighbors(result.FeatureVec)
// 		if err != nil {
// 			return 0.0, err
// 		}
// 		prediction = nil
// 		for _, neighborID := range neighborsIDs {
// 			prediction = append(prediction, neighborID)
// 		}
// 		averageRecall += Recall(prediction, result.NeighborsIds)
// 	}
// 	return averageRecall / float64(len(results)), nil
// }

// // Validate takes the array of distance thresholds and returns array of recall values
// func (benchClient *BenchClient) Validate(thrshs []float64) ([]float64, error) {
// 	metrics := make([]float64, len(thrshs))
// 	results, err := db.GetDbRecords(benchClient.TestCollection, db.FindQuery{Proj: bson.M{"featureVec": 1}})
// 	if err != nil {
// 		return nil, err
// 	}
// 	for _, thrsh := range thrshs {
// 		start := time.Now()
// 		recall, err := benchClient.ValidateThrsh(results, thrsh)
// 		if err != nil {
// 			return nil, err
// 		}
// 		metrics = append(metrics, recall)
// 		elapsed := time.Since(start)
// 		benchClient.Logger.Info.Printf("Elapsed time: %v; Thrsh: %v; Recall: %v", elapsed, thrsh, recall)
// 	}
// 	return metrics, nil
// }

// // Populate put vectors into search index
// func (benchClient *BenchClient) PopulateDataset(batchSize int, dataCollName string) error {
// 	dataColl := benchClient.Mongo.GetCollection(dataCollName)
// 	convMean, convStd, err := db.GetAggregatedStats(dataColl)
// 	if err != nil {
// 		return err
// 	}

// 	benchClient.Logger.Info.Println(convMean) // DEBUG - check for not being [0]
// 	benchClient.Logger.Info.Println(convStd)  // DEBUG - check for not being [0]

// 	benchClient.Client.BuildHasher(convMean, convStd)

// 	cursor, err := dataColl.GetCursor(db.FindQuery{})
// 	for cursor.Next(context.Background()) {
// 		err = benchClient.putBatch(cursor, batchSize)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// // putBatch accumulates db documents in a batch of desired length and calculates hashes
// func (benchClient *BenchClient) putBatch(cursor *mongo.Cursor, batchSize int) error {
// 	batch := make([]vc.RequestData, batchSize)
// 	batchID := 0
// 	for cursor.Next(context.Background()) {
// 		var record db.VectorRecord
// 		if err := cursor.Decode(&record); err != nil {
// 			continue
// 		}
// 		batch[batchID] = vc.RequestData{
// 			SecondaryID: record.SecondaryID,
// 			Vec:         record.FeatureVec,
// 		}
// 		batchID++
// 	}
// 	err := benchClient.Client.PutHashes(batch[:batchID])
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
