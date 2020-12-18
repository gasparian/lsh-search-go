package app

import (
	cm "vector-search-go/common"
	"vector-search-go/db"
	hashing "vector-search-go/lsh"
)

// Config holds general constants
type Config struct {
	BatchSize       int
	MaxHashesNumber int
	MaxNN           int
	DistanceThrsh   float64
}

// ServiceConfig holds all needed variables to run the app
type ServiceConfig struct {
	Hasher hashing.Config
	Db     db.Config
	App    Config
}

// ANNServer holds Hasher itself and the mongo Client
type ANNServer struct {
	Hasher *hashing.Hasher
	Mongo  db.MongoDatastore
	Logger *cm.Logger
	Config ServiceConfig
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
	Neighbors []ResponseRecord `json:"neighbors"`
}
