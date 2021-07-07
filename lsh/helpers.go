package lsh

import (
	"errors"
	"fmt"
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
			sample[i] = rand.Intn(len(data))
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
func GetMeanStdSampledRecords(vecs [][]float64, sampleSize int) ([]float64, []float64, error) {
	if len(vecs) == 0 {
		return nil, nil, dataSliceEmptyErr
	}
	if sampleSize <= 0 {
		return nil, nil, sampleSizeErr
	}
	sample := make([]int, sampleSize)
	if len(vecs) <= sampleSize {
		sampleSize = len(vecs)
		for i := 0; i < sampleSize; i++ {
			sample[i] = i
		}
	} else {
		for i := 0; i < sampleSize; i++ {
			sample[i] = rand.Intn(len(vecs))
		}
	}
	sampleSizeF := float64(sampleSize)
	vecLen := len(vecs[0])
	mean := mat.NewVecDense(vecLen, nil)
	for _, idx := range sample {
		mean.AddVec(mean, mat.NewVecDense(vecLen, vecs[idx]))
	}
	mean.ScaleVec(1/sampleSizeF, mean)
	std := make([]float64, len(vecs[0]))
	for _, idx := range sample {
		for j, val := range vecs[idx] {
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
type L2 bool

func NewL2() L2 {
	return L2(false)
}
func (l2 L2) GetDist(l, r []float64) float64 {
	lBlas := NewVec(l)
	rBlas := NewVec(r)
	res := NewVec(make([]float64, rBlas.N))
	blas64.Copy(rBlas, res)
	blas64.Axpy(-1.0, lBlas, res)
	return blas64.Nrm2(res)
}

func (l2 L2) IsAngular() bool {
	return bool(l2)
}

// StandartScaler ...
type StandartScaler struct {
	sync.RWMutex
	mean *mat.VecDense
	std  *mat.VecDense
}

func checkConvertVec(inp []float64, fill float64, nDims int) blas64.Vector {
	inpVecInternal := NewVec(make([]float64, nDims))
	if inp != nil && len(inp) == nDims {
		inpVecInternal.Data = inp
		copy(inpVecInternal.Data, inp)
		return inpVecInternal
	}
	if fill > 0 {
		for i := range inpVecInternal.Data {
			inpVecInternal.Data[i] = fill
		}
	}
	return inpVecInternal
}

func NewStandartScaler(mean, std []float64, nDims int) *StandartScaler {
	scaler := &StandartScaler{}
	scaler.mean = mat.NewVecDense(len(mean), nil)
	scaler.mean.SetRawVector(checkConvertVec(mean, 0.0, nDims))
	scaler.std = mat.NewVecDense(len(mean), nil)
	scaler.std.SetRawVector(checkConvertVec(std, 1.0, nDims))
	return scaler
}

func (s *StandartScaler) Scale(vec []float64) blas64.Vector {
	s.RLock()
	defer s.RUnlock()
	cpy := make([]float64, len(vec))
	copy(cpy, vec)
	res := mat.NewVecDense(len(cpy), cpy)
	res.AddScaledVec(res, -1.0, s.mean)
	res.DivElemVec(res, s.std)
	return res.RawVector()
}

// Angular calculates cosine distance between two given vectors
type Angular bool

func NewAngular() Angular {
	return Angular(true)
}

// NOTE: using Euclidean distance of normalized vectors as angular distance: sqrt(2(1-cos(u,v)))
// func (c Angular) GetDist(l, r []float64) float64 {
// 	lBlas := NewVec(l)
// 	rBlas := NewVec(r)
// 	lNorm := blas64.Nrm2(lBlas)
// 	rNorm := blas64.Nrm2(rBlas)
// 	var dist float64 = 2.0
// 	lrNorm := lNorm * rNorm
// 	if lrNorm > tol {
// 		cosine := blas64.Dot(lBlas, rBlas) / lrNorm
// 		dist = 2.0 - 2.0*cosine
// 	}
// 	if dist < tol {
// 		return 0.0
// 	}
// 	return math.Sqrt(dist)
// }

// NOTE: just regular cosine distance
func (c Angular) GetDist(l, r []float64) float64 {
	lBlas := NewVec(l)
	rBlas := NewVec(r)
	lNorm := blas64.Nrm2(lBlas)
	rNorm := blas64.Nrm2(rBlas)
	var dist float64 = 1.0
	lrNorm := lNorm * rNorm
	if lrNorm > tol {
		cosine := blas64.Dot(lBlas, rBlas) / lrNorm
		dist = 1.0 - cosine
	}
	if dist < tol {
		return 0.0
	}
	return dist
}

func (c Angular) IsAngular() bool {
	return bool(c)
}

func AngularToCosineDist(angular float64) float64 {
	return (angular * angular) / 2
}

func CosineDistToAngular(cosine float64) float64 {
	return math.Sqrt(2 * cosine)
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

func getBucketName(perm int, hash uint64) string {
	return fmt.Sprintf("%v_%v", perm, hash)
}
