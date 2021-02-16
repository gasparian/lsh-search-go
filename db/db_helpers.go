package db

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConvertAggResult makes Vector from the bson from Mongo
func ConvertAggResult(inp interface{}) ([]float64, error) {
	val, ok := inp.(primitive.A)
	if !ok {
		return nil, errors.New("type conversion failed")
	}
	conv := make([]float64, len(val))
	for i := range conv {
		v, ok := val[i].(float64)
		if !ok {
			return nil, errors.New("type conversion failed")
		}
		conv[i] = v
	}
	return conv, nil
}

// GetAggregatedStats returns vectors with Mongo aggregation results (mean and std vectors)
func GetAggregatedStats(coll MongoCollection) ([]float64, []float64, error) {
	results, err := coll.GetAggregation(GroupMeanStd)
	if err != nil {
		return nil, nil, err
	}
	convMean, err := ConvertAggResult(results[0]["avg"])
	if err != nil {
		return nil, nil, err
	}
	convStd, err := ConvertAggResult(results[0]["std"])
	if err != nil {
		return nil, nil, err
	}
	return convMean, convStd, nil
}

// GetDbRecords get documents from the db collection by field and query (aka `find`)
func GetDbRecords(coll MongoCollection, query FindQuery) ([]VectorRecord, error) {
	cursor, err := coll.GetCursor(query)
	if err != nil {
		return nil, err
	}
	var results []VectorRecord
	err = cursor.All(context.Background(), &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}
