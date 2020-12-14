package app

import (
	"errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"context"
	"log"
	"sort"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	cm "vector-search-go/common"
	"vector-search-go/db"
)

// NewANNServer returns empty index object with initialized mongo client
func NewANNServer() (ANNServer, error) {
	mongodb, err := db.GetDbClient(dbLocation)
	if err != nil {
		log.Println("Creating db client: " + err.Error())
		return ANNServer{}, err
	}
	// defer mongodb.Disconnect() // should be placed on the upper level

	searchHandler := ANNServer{
		MongoClient: *mongodb,
	}
	err = searchHandler.LoadIndexer()
	if err != nil {
		return ANNServer{}, err
	}
	return searchHandler, nil
}

// LoadIndexer load indexer from the db if it exists
func (annServer *ANNServer) LoadIndexer() error {
	database := annServer.MongoClient.GetDb(dbName)
	helperColl := database.Collection(helperCollectionName)
	result, err := db.GetHelperRecord(helperColl)
	if err != nil {
		return err
	}
	if len(result.Indexer) > 0 && result.IsBuildDone {
		annServer.Index.Load(result.Indexer)
		return nil
	}
	return errors.New("Can't load indexer object")
}

// hashDbRecordsBatch accumulates db documents in a batch of desired length and calculates hashes
func (annServer *ANNServer) hashDbRecordsBatch(cursor *mongo.Cursor, batchSize int) ([]interface{}, error) {
	batch := make([]interface{}, batchSize)
	batchID := 0
	for cursor.Next(context.Background()) {
		var record db.VectorRecord
		if err := cursor.Decode(&record); err != nil {
			continue
		}
		hashes, err := annServer.Index.GetHashes(
			cm.Vector{
				Values: record.FeatureVec,
				Size:   len(record.FeatureVec),
			},
		)
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

// popHashRecord drops record from collection by objectID (string Hex)
func (annServer *ANNServer) popHashRecord(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	database := annServer.MongoClient.GetDb(dbName)
	helperRecord, err := db.GetHelperRecord(database.Collection(helperCollectionName))
	if err != nil {
		return err
	}
	hashesColl := database.Collection(helperRecord.HashCollName)
	err = db.DeleteRecords(hashesColl, bson.D{{"_id", objectID}})
	if err != nil {
		return err
	}
	return nil
}

// putHashRecord drops record from collection by objectID (string Hex)
func (annServer *ANNServer) putHashRecord(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	database := annServer.MongoClient.GetDb(dbName)
	helperRecord, err := db.GetHelperRecord(database.Collection(helperCollectionName))
	if err != nil {
		return err
	}
	hashesColl := database.Collection(helperRecord.HashCollName)
	records, err := db.GetDbRecords(
		database.Collection(dataCollectionName),
		db.FindQuery{
			Limit: 1,
			Query: bson.D{{"_id", objectID}},
		},
	)
	if err != nil {
		return err
	}
	recordInterfaces := make([]interface{}, len(records))
	for i := range records {
		recordInterfaces[i] = records[i]
	}
	err = db.SetRecords(hashesColl, recordInterfaces)
	if err != nil {
		return err
	}
	return nil
}

// getNeighbors returns filtered nearest neighbors sorted by distance in ascending order
func (annServer *ANNServer) getNeighbors(input RequestData) (*ResponseData, error) {
	inputVec := cm.NewVector(input.Vec)
	var hashesRecords []db.HashesRecord
	database := annServer.MongoClient.GetDb(dbName)
	if input.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(input.ID)
		if err != nil {
			return nil, err
		}
		helperRecord, err := db.GetHelperRecord(database.Collection(helperCollectionName))
		if err != nil {
			return nil, err
		}
		hashesColl := database.Collection(helperRecord.HashCollName)
		hashesRecords, err = db.GetHashesRecords(
			hashesColl,
			db.FindQuery{
				Limit: 1,
				Query: bson.D{{"_id", objectID}},
				Proj:  bson.M{"_id": 1},
			},
		)
		if err != nil {
			return nil, err
		}
	} else if !inputVec.IsZero() {
		hashes, err := annServer.Index.GetHashes(inputVec)
		helperRecord, err := db.GetHelperRecord(database.Collection(helperCollectionName))
		if err != nil {
			return nil, err
		}
		hashesColl := database.Collection(helperRecord.HashCollName)
		hashesQuery := bson.D{}
		for k, v := range hashes {
			hashesQuery = append(hashesQuery, bson.E{strconv.Itoa(k), v})
		}
		hashesRecords, err = db.GetHashesRecords(
			hashesColl,
			db.FindQuery{
				Limit: maxHashesNo,
				Query: hashesQuery,
				Proj:  bson.M{"_id": 1},
			},
		)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("Get NN: object ID or vector must be provided")
	}

	vectorIDs := bson.A{}
	for idx := range hashesRecords {
		vectorIDs = append(vectorIDs, hashesRecords[idx].ID)
	}
	hashesRecords = nil

	vectorsCursor, err := db.GetCursor(
		database.Collection(dataCollectionName),
		db.FindQuery{
			Query: bson.D{{"_id", bson.D{{"$in", vectorIDs}}}},
			Proj:  bson.M{"_id": 1, "featureVec": 1},
		},
	)
	if err != nil {
		return nil, err
	}

	neighbors := make([]ResponseRecord, maxNN)
	var idx int = 0
	var candidate db.VectorRecord
	for vectorsCursor.Next(context.Background()) && idx <= maxNN {
		if err := vectorsCursor.Decode(&candidate); err != nil {
			continue
		}
		hexID := candidate.ID.Hex()
		dist := annServer.Index.GetDist(inputVec, cm.NewVector(candidate.FeatureVec))
		if dist <= distanceThrsh {
			neighbors[idx] = ResponseRecord{
				ID:   hexID,
				Dist: dist,
			}
			idx++
		}
	}
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].Dist < neighbors[j].Dist
	})
	return &ResponseData{NN: neighbors}, nil
}
