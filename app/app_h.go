package app

import (
	cm "lsh-search-engine/common"
	"lsh-search-engine/db"
	hashing "lsh-search-engine/lsh"
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
	Hasher        *hashing.Hasher
	Mongo         db.MongoDatastore
	Logger        *cm.Logger
	Config        ServiceConfig
	LastBuildTime int64
}
