package common

import (
	"log"
)

// Used to represent the hasher build status
const (
	BuildStatusUnknown = iota
	BuildStatusError
	BuildStatusInProgress
	BuildStatusDone
)

// Logger holds several logger instances with different prefixes
type Logger struct {
	Warn *log.Logger
	Info *log.Logger
	Err  *log.Logger
}

// NeighborsRecord holds a single neighbor
type NeighborsRecord struct {
	ID          string  `json:"id,omitempty"`
	SecondaryID int     `json:"secondaryId,omitempty"`
	Dist        float64 `json:"dist,omitempty"`
}

// ResponseData holds the response data of any hanlder
type ResponseData struct {
	Results interface{} `json:"neighbors,omitempty"`
	Message string      `json:"message,omitempty"`
}

// RequestData used for unpacking the request payload for Pop/Put vectors
type RequestData struct {
	ID          string    `json:"id,omitempty"`
	SecondaryID int       `json:"secondaryId,omitempty"`
	Vec         []float64 `json:"vec,omitempty"`
}

// DatasetStats holds basic feature vector stats like mean and standart deviation
type DatasetStats struct {
	Mean []float64 `json:"mean"`
	Std  []float64 `json:"std"`
}
