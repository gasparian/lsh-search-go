package db

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	cm "vector-search-go/common"
)

var (
	dbtimeOut, _ = strconv.Atoi(os.Getenv("DB_CLIENT_TIMEOUT"))
)

// GetDbClient creates client for talking to mongodb
func GetDbClient(dbLocation string) (*MongoClient, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(dbLocation))
	if err != nil {
		return nil, err
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Duration(dbtimeOut)*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}
	mongodb := &MongoClient{
		Ctx:    ctx,
		Client: client,
	}
	return mongodb, nil
}

// Disconnect client from the context
func (mongodb *MongoClient) Disconnect() {
	mongodb.Client.Disconnect(mongodb.Ctx)
}

// GetDb returns database object
func (mongodb *MongoClient) GetDb(dbName string) *mongo.Database {
	return mongodb.Client.Database(dbName)
}

// GetAggregation runs prepared aggregation pipeline in mongodb
func GetAggregation(coll *mongo.Collection, groupStage mongo.Pipeline) ([]bson.M, error) {
	opts := options.Aggregate().SetMaxTime(time.Duration(dbtimeOut) * time.Second)
	cursor, err := coll.Aggregate(context.TODO(), groupStage, opts)
	if err != nil {
		return nil, err
	}

	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}
	return results, nil
}

// ConvertAggResult makes Vector from the bson from Mongo
func ConvertAggResult(inp interface{}) (cm.Vector, error) {
	val, ok := inp.(primitive.A)
	if !ok {
		return cm.Vector{}, errors.New("Type conversion failed")
	}
	conv := cm.Vector{
		Values: make([]float64, len(val)),
		Size:   len(val),
	}
	for i := range conv.Values {
		v, ok := val[i].(float64)
		if !ok {
			return cm.Vector{}, errors.New("Type conversion failed")
		}
		conv.Values[i] = v
	}
	return conv, nil
}

// GetAggregatedStats returns vectors with Mongo aggregation results (mean and std vectors)
func GetAggregatedStats(coll *mongo.Collection) (cm.Vector, cm.Vector, error) {
	results, err := GetAggregation(coll, GroupMeanStd)
	if err != nil {
		log.Println("Making db aggregation: " + err.Error())
		return cm.Vector{}, cm.Vector{}, err
	}
	convMean, err := ConvertAggResult(results[0]["avg"])
	if err != nil {
		log.Println("Parsing aggregation result: " + err.Error())
		return cm.Vector{}, cm.Vector{}, err
	}
	convStd, err := ConvertAggResult(results[0]["std"])
	if err != nil {
		log.Println("Parsing aggregation result: " + err.Error())
		return cm.Vector{}, cm.Vector{}, err
	}
	return convMean, convStd, nil
}

// UpdateCollection gets documents from the db,
// update these documents with the new fields
// TO DO
func UpdateCollection() {
}

// SetField updates the selected field of the doc.
// TO DO
func SetField() {

}

// SetData adds the new document in the collection
// TO DO
func SetData() {

}

// GetData get documents from db by field and query
// TO DO
func GetData() {

}

// MakeHelperCollection checks if the helper collection exists
// in the db, and creates them if needed; helper collection stores
// any data for synchronizing the search index state
// TO DO
func MakeHelperCollection() {
}
