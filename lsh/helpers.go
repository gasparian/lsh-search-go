package lsh

import (
	"errors"
	"gonum.org/v1/gonum/blas/blas64"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"sync"
)

const (
	tol = 1e-6
)

var (
	dataSliceEmptyErr = errors.New("Data slice is empty")
	sampleSizeErr     = errors.New("Sample size must be > 0")
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

func GenerateRandomInt(min, max int) int {
	return rand.Intn(max-min) + min
}

// GetMeanStd returns mean and std based on incoming NxM matrix
func GetMeanStdSampled(data [][]float64, sampleSize int) ([]float64, []float64, error) {
	if len(data) == 0 {
		return nil, nil, dataSliceEmptyErr
	}
	if sampleSize <= 0 {
		return nil, nil, sampleSizeErr
	}
	sample := make([]int, sampleSize)
	if len(data) <= sampleSize {
		sampleSize = len(data)
		for i := 0; i < sampleSize; i++ {
			sample[i] = i
		}
	} else {
		for i := 0; i < sampleSize; i++ {
			sample[i] = GenerateRandomInt(0, len(data))
		}
	}
	sampleSizeF := float64(sampleSize)
	vecLen := len(data[0])
	mean := mat.NewVecDense(vecLen, nil)
	for _, idx := range sample {
		mean.AddVec(mean, mat.NewVecDense(vecLen, data[idx]))
	}
	mean.ScaleVec(1/sampleSizeF, mean)
	std := make([]float64, len(data[0]))
	for _, idx := range sample {
		for j, val := range data[idx] {
			shifted := val - mean.AtVec(j)
			std[j] += math.Sqrt(shifted * shifted)
		}
	}
	stdVec := mat.NewVecDense(vecLen, std)
	stdVec.ScaleVec(1/sampleSizeF, stdVec)
	return mean.RawVector().Data, stdVec.RawVector().Data, nil
}

// GetMeanStdSampledRecords duplucate of GetMeanStdSample but for the Record data type, must be refactored later
func GetMeanStdSampledRecords(data []Record, sampleSize int) ([]float64, []float64, error) {
	if len(data) == 0 {
		return nil, nil, dataSliceEmptyErr
	}
	if sampleSize <= 0 {
		return nil, nil, sampleSizeErr
	}
	sample := make([]int, sampleSize)
	if len(data) <= sampleSize {
		sampleSize = len(data)
		for i := 0; i < sampleSize; i++ {
			sample[i] = i
		}
	} else {
		for i := 0; i < sampleSize; i++ {
			sample[i] = GenerateRandomInt(0, len(data))
		}
	}
	sampleSizeF := float64(sampleSize)
	vecLen := len(data[0].Vec)
	mean := mat.NewVecDense(vecLen, nil)
	for _, idx := range sample {
		mean.AddVec(mean, mat.NewVecDense(vecLen, data[idx].Vec))
	}
	mean.ScaleVec(1/sampleSizeF, mean)
	std := make([]float64, len(data[0].Vec))
	for _, idx := range sample {
		for j, val := range data[idx].Vec {
			shifted := val - mean.AtVec(j)
			std[j] += math.Sqrt(shifted * shifted)
		}
	}
	stdVec := mat.NewVecDense(vecLen, std)
	stdVec.ScaleVec(1/sampleSizeF, stdVec)
	return mean.RawVector().Data, stdVec.RawVector().Data, nil
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

// StandartScaler ...
type StandartScaler struct {
	sync.RWMutex
	mean *mat.VecDense
	std  *mat.VecDense
}

func NewStandartScaler(mean, std []float64) *StandartScaler {
	return &StandartScaler{
		mean: mat.NewVecDense(len(mean), mean),
		std:  mat.NewVecDense(len(mean), std),
	}
}

func (s *StandartScaler) Scale(vec []float64) []float64 {
	s.RLock()
	defer s.RUnlock()
	res := mat.NewVecDense(len(vec), nil)
	res.AddScaledVec(mat.NewVecDense(len(vec), vec), -1.0, s.mean)
	res.DivElemVec(res, s.std)
	return res.RawVector().Data
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
