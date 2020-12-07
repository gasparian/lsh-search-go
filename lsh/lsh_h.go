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
	Dims       int
	Bias       float64
	MaxNPlanes int
	MeanVec    cm.Vector
	MaxDist    float64
	Planes     []Plane
	nPlanes    int
}

// LSHIndex holds N_PERMUTS number of LSHIndexInstance instances
type LSHIndex struct {
	sync.Mutex
	Entries []LSHIndexInstance
}
