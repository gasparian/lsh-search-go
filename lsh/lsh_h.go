package lsh

import (
	"sync"

	cm "vector-search-go/common"
)

// Plane struct holds data needed to work with plane
type Plane struct {
	Coefs      cm.Vector
	InnerPoint cm.Vector
}

// HasherInstance holds data for local sensetive hashing algorithm
type HasherInstance struct {
	Dims    int
	Bias    float64
	MeanVec cm.Vector
	MaxDist float64
	Planes  []Plane
	NPlanes int
}

// Config holds all needed constants for creating the Hasher instance
type Config struct {
	IsAngularDistance int
	MaxNPlanes        int
	NPermutes         int
}

// Hasher holds N_PERMUTS number of HasherInstance instances
type Hasher struct {
	sync.Mutex
	Config          Config
	Instances       []HasherInstance
	HashFieldsNames []string
}

// HasherEncode using for encoding/decoding the Hasher structure
type HasherEncode struct {
	Instances *[]HasherInstance
}
