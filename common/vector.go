package common

import (
	"gonum.org/v1/gonum/blas/blas64"
)

// NewVec creates new blas vector
func NewVec(data []float64) blas64.Vector {
	if data == nil {
		data = make([]float64, 0)
	}
	return blas64.Vector{
		N:    len(data),
		Inc:  1,
		Data: data,
	}
}

// L2 calculates l2-distance between two vectors
func L2(a, b blas64.Vector) float64 {
	var res blas64.Vector
	blas64.Copy(b, res)
	blas64.Axpy(-1.0, a, res)
	return blas64.Nrm2(res)
}

// CosineSim calculates cosine similarity of the two given vectors
func CosineSim(a, b blas64.Vector) float64 {
	cosine := blas64.Dot(a, b) / (blas64.Nrm2(a) * blas64.Nrm2(b))
	return 1.0 - cosine
}
