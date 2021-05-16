package lsh

import (
	"errors"
	"gonum.org/v1/gonum/blas/blas64"
	"math"
	"math/rand"
	"sync"
)

const (
	tol = 1e-6
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
func GetMeanStd(data []Record, sampleSize int) ([]float64, []float64, error) {
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
	mean := make([]float64, len(data[0].Vec))
	for _, idx := range sample {
		for j, val := range data[idx].Vec {
			mean[j] += val / sampleSizeF
		}
	}
	std := make([]float64, len(data[0].Vec))
	for _, idx := range sample {
		for j, val := range data[idx].Vec {
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
type L2 int

func NewL2() L2 {
	return L2(42)
}
func (l2 L2) GetDist(l, r []float64) float64 {
	lBlas := NewVec(l)
	rBlas := NewVec(r)
	res := NewVec(make([]float64, rBlas.N))
	blas64.Copy(rBlas, res)
	blas64.Axpy(-1.0, lBlas, res)
	return blas64.Nrm2(res)
}

// IsZeroVectorBlas returns true if the sum of vectors' elements close to 0.0
func IsZeroVectorBlas(v blas64.Vector) bool {
	return math.Abs(blas64.Asum(v)) <= tol
}

// Cosine calculates cosine distance between two given vectors
type Cosine int

func NewCosine() Cosine {
	return Cosine(42)
}
func (c Cosine) GetDist(l, r []float64) float64 {
	lBlas := NewVec(l)
	rBlas := NewVec(r)
	if IsZeroVectorBlas(lBlas) || IsZeroVectorBlas(rBlas) {
		return 1.0 // NOTE: zero vectors are wrong with angular metric
	}
	cosine := blas64.Dot(lBlas, rBlas) / (blas64.Nrm2(lBlas) * blas64.Nrm2(rBlas))
	return 1.0 - cosine
}

type StringSet struct {
	mx    sync.RWMutex
	Items map[string]bool
}

func NewStringSet() *StringSet {
	return &StringSet{
		Items: make(map[string]bool),
	}
}

func (s *StringSet) Get(key string) bool {
	s.mx.RLock()
	defer s.mx.RUnlock()
	_, ok := s.Items[key]
	return ok
}

func (s *StringSet) Set(key string) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.Items[key] = true
}

func (s *StringSet) Remove(key string) {
	s.mx.Lock()
	defer s.mx.Unlock()
	delete(s.Items, key)
}
