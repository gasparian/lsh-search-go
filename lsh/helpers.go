package lsh

import (
	"errors"
	"gonum.org/v1/gonum/blas/blas64"
	"math"
	"math/rand"
)

const tol = 1e-6

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

// ConvertToInt __
func ConvertToInt(ar []int32) []int {
	newar := make([]int, len(ar))
	var v int32
	var i int
	for i, v = range ar {
		newar[i] = int(v)
	}
	return newar
}

func generateRandomInt(min, max int) int {
	return rand.Intn(max-min) + min
}

// GetMeanStd returns mean and std based on incoming NxM matrix
func GetMeanStd(data [][]float64, sampleSize int) ([]float64, []float64, error) {
	if len(data) == 0 {
		return nil, nil, errors.New("Data slice is empty")
	}
	if sampleSize <= 0 {
		return nil, nil, errors.New("sampleSize must be > 0")
	}
	if len(data) <= sampleSize {
		sampleSize = len(data)
	}
	sample := make([]int, sampleSize)
	sampleSizeF := float64(sampleSize)
	for i := 0; i < sampleSize; i++ {
		sample[i] = generateRandomInt(0, len(data))
	}
	mean := make([]float64, len(data[0]))
	for _, idx := range sample {
		for j, val := range data[idx] {
			mean[j] += val / sampleSizeF
		}
	}
	std := make([]float64, len(data[0]))
	for _, idx := range sample {
		for j, val := range data[idx] {
			std[j] += math.Sqrt((val-mean[j])*(val-mean[j])) / sampleSizeF
		}
	}
	return mean, std, nil
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

// IsZeroVectorBlas returns true if the sum of vectors' elements close to 0.0
func IsZeroVectorBlas(v blas64.Vector) bool {
	return math.Abs(blas64.Asum(v)) <= tol
}

// IsZeroVector __
func IsZeroVector(v []float64) bool {
	var sum float64 = 0.0
	for _, val := range v {
		sum += val
	}
	return math.Abs(sum) <= tol
}
