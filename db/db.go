package db

import (
	"context"
	// "log"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
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

// Disconnects client from the context
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
	// cursor, err := coll.Aggregate(context.TODO(), mongo.Pipeline{groupStage}, opts)
	cursor, err := coll.Aggregate(context.TODO(), groupStage, opts)
	if err != nil {
		return nil, err
	}

	// get a list of all returned documents and print them out
	// see the mongo.Cursor documentation for more examples of using cursors
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}
	return results, nil
}
