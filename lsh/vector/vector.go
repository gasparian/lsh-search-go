package vector

import (
	"errors"
	"gonum.org/v1/gonum/blas/blas64"
	"math"
)

// ConvertTo64 __
func ConvertTo64(ar []float32) []float64 {
	newar := make([]float64, len(ar))
	var v float32
	var i int
	for i, v = range ar {
		newar[i] = float64(v)
	}
	return newar
}

// GetStat returns mean of vector or std, if the bias argument is not empty
// TODO: optimize with goroutines and blas64; add test
func GetStat(data [][]float64, bias []float64, tol float64, maxSampleSize int) ([]float64, error) {
	if len(data) == 0 {
		return nil, errors.New("Data slice is empty")
	}
	mean := make([]float64, len(data[0]))
	var relativeDiff, count, val float64
	var meanRelativeDiff float64 = 1.0
	var isStd bool = false
	if !IsZeroVector(NewVec(bias)) {
		isStd = true
	}
	for i := 0; i < len(data); i++ {
		if i >= maxSampleSize || meanRelativeDiff <= tol {
			count = (float64)(i) + 1
			break
		}
		oldMeanVec := mean
		meanRelativeDiff = 0.0
		for j := 0; j < len(mean); j++ {
			val = data[i][j]
			if isStd {
				val = math.Sqrt((val - bias[j]) * (val - bias[j]))
			}
			mean[j] += val
			relativeDiff = oldMeanVec[j] / (mean[j] / (float64)(i+1))
			meanRelativeDiff += relativeDiff / (float64)(len(mean))
		}
	}
	for i := range mean {
		mean[i] /= count
	}
	return mean, nil
}

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
	res := NewVec(b.Data)
	blas64.Axpy(-1.0, a, res)
	return blas64.Nrm2(res)
}

// CosineSim calculates cosine similarity btw the two given vectors
func CosineSim(a, b blas64.Vector) float64 {
	cosine := blas64.Dot(a, b) / (blas64.Nrm2(a) * blas64.Nrm2(b))
	return 1.0 - cosine
}

// IsZeroVector determines if vector zero or not
func IsZeroVector(v blas64.Vector) bool {
	return blas64.Asum(v) == 0.0
}
