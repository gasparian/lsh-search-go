package app

import (
	"vector-search-go/db"
	hasher "vector-search-go/lsh"
)

var (
	// HelloMessage just holds message which describes the public api
	HelloMessage = getHelloMessage()
)

// ANNServer holds Indexer itself and the mongo Client
type ANNServer struct {
	Index       *hasher.LSHIndex
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
