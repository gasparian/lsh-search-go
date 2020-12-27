package app

import (
	cm "lsh-search-service/common"
	"lsh-search-service/db"
	hashing "lsh-search-service/lsh"
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

// DatasetStats holds basic feature vector stats like mean and standart deviation
type DatasetStats struct {
	Mean []float64 `json:"mean"`
	Std  []float64 `json:"std"`
}
