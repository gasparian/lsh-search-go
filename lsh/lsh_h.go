package lsh

import (
	"sync"

	"gonum.org/v1/gonum/blas/blas64"
)

// Plane struct holds data needed to work with plane
type Plane struct {
	Coefs      blas64.Vector
	InnerPoint blas64.Vector
}

// HasherInstance holds data for local sensetive hashing algorithm
type HasherInstance struct {
	Dims    int
	Bias    float64
	MeanVec blas64.Vector
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
