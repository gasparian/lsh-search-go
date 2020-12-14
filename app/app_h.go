package app

import (
	"vector-search-go/db"
	alg "vector-search-go/lsh"
)

var (
	// HelloMessage just holds message which describes the public api
	// TO DO: update the API reference
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

// RequestData used for unpacking the request payload for Pop/Put vectors
type RequestData struct {
	ID  string    `json:"id,omitempty"`
	Vec []float64 `json:"vec,omitempty"`
}

// ResponseRecord holds a single neighbor
type ResponseRecord struct {
	ID   string  `json:"id"`
	Dist float64 `json:"dist"`
}

// ResponseData holds the resulting objectIDs of nearest neighbors found
type ResponseData struct {
	NN []ResponseRecord `json:"neighbors"`
}
