package algorithm

import (
	"sync"
)

// IVector here just to quicly observe which methods exists on Vector struct
type IVector interface {
	Add(Vector) Vector
	ConstMul(float64) Vector
	DotProd(Vector) float64
	L2(Vector) float64
	CosineSim(Vector) float64
}

// Vector is basic data structure to hold slice of floats and it's size
type Vector struct {
	Values []float64
	Size   int
}

// Indexer basic interface that should implement any indexer object
type Indexer interface {
	Build() error
	GetHash(*Vector) uint64
}

// Plane struct holds data needed to work with plane
type Plane struct {
	Coefs      Vector
	InnerPoint Vector
}

// LSHIndexRecord holds data for local sensetive hashing algorithm
type LSHIndexRecord struct {
	Dims       int
	Bias       float64
	MaxNPlanes int
	MeanVec    Vector
	MaxDist    float64
	Planes     []Plane
	nPlanes    int
}

// LSHIndex holds N_PERMUTS number of LSHIndexRecord instances
type LSHIndex struct {
	sync.Mutex
	Entries []LSHIndexRecord
}
