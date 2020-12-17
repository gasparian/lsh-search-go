package app

import (
	"log"
	"vector-search-go/db"
	hashing "vector-search-go/lsh"
)

// AppConfig holds general constants
type AppConfig struct {
	DbLocation           string
	DbName               string
	DataCollectionName   string
	HelperCollectionName string
	BatchSize            int
	MaxHashesNumber      int
	MaxNN                int
	DistanceThrsh        float64
}

// Config holds all needed variables to run the app
type Config struct {
	Hasher hashing.LSHConfig
	App    AppConfig
}

// Logger holds several logger instances with different prefixes
type Logger struct {
	Warn  *log.Logger
	Info  *log.Logger
	Build *log.Logger
	Err   *log.Logger
}

// ANNServer holds Indexer itself and the mongo Client
type ANNServer struct {
	Index       *hashing.LSHIndex
	MongoClient db.MongoClient
	Logger      Logger
	Config      AppConfig
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
