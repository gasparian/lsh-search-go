package app

import (
	"vector-search-go/db"
	alg "vector-search-go/lsh"
)

var (
	// HelloMessage just holds message which describes the public api
	HelloMessage = []byte(`{
		"methods": {
			"GET": {
				"/build": "starts building search index from scratch; returns task id, which could be queried later",
				"/checkBuild?Key=<BUILD_TASK_ID>": "returns status of build task by unique id",
				"/pop?id=<POINT_ID>": "removes the point from search index (drops the hashes field in a document)"
			},
			"POST": {
				"/set": "add vector to the search index (and db, if it's not there yet)",
				"/get": "returns db ids of the nearest points"
			}
	    }
	}`)
)

// ANNServer holds Indexer itself and the mongo Client
type ANNServer struct {
	Index       *alg.LSHIndex
	MongoClient db.MongoClient
}
