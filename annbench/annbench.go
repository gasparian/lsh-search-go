package annbench

import (
	"context"
	"os"
	// cm "lsh-search-engine/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	cl "lsh-search-engine/client"
	"lsh-search-engine/db"
)

var (
	testCollectionName = os.Getenv("TEST_COLLECTION_NAME")
)

// BenchClient holds db for getting vectors from test collection
// and a client for performing requests to the running ann service
type BenchClient struct {
	Client cl.ANNClient
	Db     db.MongoDatastore
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
// TO DO: add annClient here
func ValidateThrsh(cursor *mongo.Cursor, thrsh float64) (float64, error) {
	var queryVector db.VectorRecord
	for cursor.Next(context.Background()) {
		if err := cursor.Decode(&queryVector); err != nil {
			return 0.0, err
		}
		// otherwise - call getNeighbors
	}
	return 0.0, nil
}

// Validate takes the array of distance thresholds and returns array of recall values
func Validate(config db.Config, thrshs []float64) ([]float64, error) {
	result := make([]float64, len(thrshs))
	mongodb, err := db.GetDbClient(config)
	if err != nil {
		return nil, err
	}
	defer mongodb.Disconnect()
	testColl := mongodb.GetCollection(testCollectionName)
	// TO DO: add go routines in a loop
	for _, thrsh := range thrshs {
		cursor, err := testColl.GetCursor(db.FindQuery{Proj: bson.M{"featureVec": 1}})
		if err != nil {
			return nil, err
		}
		recall, err := ValidateThrsh(cursor, thrsh)
		if err != nil {
			return nil, err
		}
		result = append(result, recall)
	}
	return result, nil
}
