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

// LSHIndexInstance holds data for local sensetive hashing algorithm
type LSHIndexInstance struct {
	Dims    int
	Bias    float64
	MeanVec cm.Vector
	MaxDist float64
	Planes  []Plane
	NPlanes int
}

// LSHConfig holds all needed constants for creating the LSHIndex instance
type LSHConfig struct {
	IsAngularDistance int
	MaxNPlanes        int
	NPermutes         int
}

// LSHIndex holds N_PERMUTS number of LSHIndexInstance instances
type LSHIndex struct {
	sync.Mutex
	Config          LSHConfig
	Instances       []LSHIndexInstance
	HashFieldsNames []string
}

// LSHIndexEncode using for encoding/decoding the LSHIndex structure
type LSHIndexEncode struct {
	Instances *[]LSHIndexInstance
}
