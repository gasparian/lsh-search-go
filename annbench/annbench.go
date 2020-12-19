package annbench

import (
	"os"
	// cm "vector-search-go/common"
	// "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"vector-search-go/db"
)

var (
	testCollectionName = os.Getenv("TEST_COLLECTION_NAME")
)

// Recall returns ratio of relevant predictions over the all true relevant items
func Recall(prediction, groundTruth []int32) float64 {
	if len(prediction) != len(groundTruth) {
		return 0.0
	}
	valid := 0
	for i := range prediction {
		if prediction[i] == groundTruth[i] {
			valid++
		}
	}
	return float64(valid) / float64(len(prediction))
}

// ValidateThrsh takes the distance threshold and returns recall value
// TO DO
func ValidateThrsh(cursor *mongo.Cursor, thrsh float64) (float64, error) {

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
	for _, thrsh := range thrshs {
		cursor, err := testColl.GetCursor(db.FindQuery{})
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
