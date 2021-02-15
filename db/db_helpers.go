package db

import (
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
