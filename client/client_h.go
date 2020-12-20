package client

import (
	"net/http"
)

// Config holds necessary constants for initiating the ANNClient
type Config struct {
	ServerAddress string
	Timeout       int
}

type methods struct {
	HealthCheck string
	CheckBuild  string
	BuildIndex  string
	GetNN       string
	PopHash     string
	PutHash     string
}

// ANNClient holds data needed to perform custom http requests
type ANNClient struct {
	ServerAddress string
	Client        http.Client
	Methods       methods
}
