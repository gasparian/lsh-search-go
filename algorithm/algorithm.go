package algorithm

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"
)

var (
	globMaxNPlanes, _ = strconv.Atoi(os.Getenv("MAX_N_PLANES"))
	globNPermutes, _  = strconv.Atoi(os.Getenv("N_PERMUTS"))
)

// NewVector creates new vector by given slice of floats
func NewVector(inpVec []float64) Vector {
	return Vector{
		Values: inpVec,
		Size:   len(inpVec),
	}
}

// Add two vectors of the same dimnsionality
func (vec *Vector) Add(rvec Vector) Vector {
	sum := NewVector(make([]float64, vec.Size))
	for i := range vec.Values {
		sum.Values[i] = vec.Values[i] + rvec.Values[i]
	}
	return sum
}

// ConstMul multiplicates vector with provided constant float
func (vec *Vector) ConstMul(constant float64) Vector {
	newVec := NewVector(make([]float64, vec.Size))
	for i := range vec.Values {
		newVec.Values[i] = vec.Values[i] * constant
	}
	return newVec
}

// DotProd calculates dot product between two vectors
func (vec *Vector) DotProd(inpVec Vector) float64 {
	var dp float64 = 0.0
	for i := range vec.Values {
		dp += vec.Values[i] * inpVec.Values[i]
	}
	return dp
}

// L2 calculates l2-distance of two vectors
func (vec *Vector) L2(inpVec Vector) float64 {
	var l2 float64
	var diff float64
	for i := range vec.Values {
		diff = vec.Values[i] - inpVec.Values[i]
		l2 += diff * diff
	}
	return math.Sqrt(l2)
}

// L2Norm calculates l2 norm of a vector
func (vec *Vector) L2Norm() float64 {
	zeroVec := Vector{
		Values: make([]float64, vec.Size),
	}
	return vec.L2(zeroVec)
}

// CosineSim calculates cosine similarity of two given vectors
func (vec *Vector) CosineSim(inpVec Vector) float64 {
	cosine := vec.DotProd(inpVec) / (vec.L2Norm() * inpVec.L2Norm())
	return cosine
}

// GetPointPlaneDist calculates distance between origin and plane
func GetPointPlaneDist(planeCoefs Vector) Vector {
	values := make([]float64, planeCoefs.Size-1)
	dCoef := planeCoefs.Values[planeCoefs.Size-1]
	var denom float64 = 0.0
	for i := range values {
		denom += planeCoefs.Values[i] * planeCoefs.Values[i]
	}
	for i := range values {
		values[i] = planeCoefs.Values[i] * dCoef / denom
	}
	return Vector{
		Values: values,
		Size:   len(values),
	}
}

// NewLSHIndexInstance creates new instance of LSHIndex object
func NewLSHIndexInstance(meanVec, stdVec Vector, maxNPlanes int) (LSHIndexInstance, error) {
	lshIndex := LSHIndexInstance{
		Dims:       meanVec.Size,
		Bias:       stdVec.L2Norm(),
		MaxNPlanes: maxNPlanes,
		MeanVec:    meanVec,
	}
	err := lshIndex.Build()
	if err != nil {
		return LSHIndexInstance{}, err
	}
	return lshIndex, nil
}

// NewLSHIndex creates slice of LSHIndexInstances to hold several permutations results
func NewLSHIndex(convMean, convStd Vector) (*LSHIndex, error) {
	lshIndex := &LSHIndex{
		Entries: make([]LSHIndexInstance, globNPermutes),
	}
	var tmpLSHIndex LSHIndexInstance
	var err error
	for i := 0; i < globNPermutes; i++ {
		tmpLSHIndex, err = NewLSHIndexInstance(convMean, convStd, globMaxNPlanes)
		if err != nil {
			return nil, err
		}
		lshIndex.Entries[i] = tmpLSHIndex
	}
	return lshIndex, nil
}

func (lsh *LSHIndexInstance) getRandomPlane() Vector {
	coefs := Vector{
		Values: make([]float64, lsh.Dims+1),
		Size:   lsh.Dims + 1,
	}
	var l2 float64 = 0.0
	for i := 0; i < lsh.Dims; i++ {
		coefs.Values[i] = -1.0 + rand.Float64()*2
		l2 += coefs.Values[i] * coefs.Values[i]
	}
	l2 = math.Sqrt(l2)
	bias := l2 * lsh.Bias
	coefs.Values[coefs.Size-1] = -1.0*bias + rand.Float64()*bias*2
	return coefs
}

// Build creates set of planes which will be used to calculate hash
func (lsh *LSHIndexInstance) Build() error {
	if lsh.Dims <= 0 {
		return errors.New("Dimensions number must be a positive integer")
	}

	rand.Seed(time.Now().UnixNano())
	var coefs Vector
	for i := 0; i < lsh.nPlanes; i++ {
		coefs = lsh.getRandomPlane()
		lsh.Planes = append(lsh.Planes, Plane{
			Coefs:      coefs,
			InnerPoint: GetPointPlaneDist(coefs),
		})
	}
	return nil
}

// GetHash calculates LSH code
func (lsh *LSHIndexInstance) GetHash(inpVec *Vector) uint64 {
	var hash uint64
	var vec Vector
	var plane *Plane
	var dpSign bool
	for i := 0; i < lsh.nPlanes; i++ {
		plane = &lsh.Planes[i]
		vec = inpVec.Add(lsh.MeanVec.ConstMul(-1.0))
		vec = vec.Add(plane.InnerPoint.ConstMul(-1.0))
		dpSign = math.Signbit(vec.DotProd(plane.Coefs))
		if !dpSign {
			hash |= (1 << i)
		}
	}
	return hash
}

// Dump writes to disk LSHIndex object as a byte-array
func (lsh *LSHIndex) Dump(path string) ([]byte, error) {
	if len(lsh.Entries) == 0 {
		return nil, errors.New("Search index must contain at least one object")
	}
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(*lsh)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Load loads LSHIndex struct into memory from byte-array file
func (lsh *LSHIndex) Load(inp []byte) error {
	buf := &bytes.Buffer{}
	buf.Write(inp)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(lsh)
	if err != nil {
		return err
	}
	return nil
}

// DumpBytesToFile writes byte array to the file
func DumpBytesToFile(inp []byte, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(inp); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return nil
}

// LoadBytesFromFile loads byte array from file
func LoadBytesFromFile(path string) ([]byte, error) {
	buf := &bytes.Buffer{}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(buf, f)
	if err != nil {
		return nil, err
	}
	f.Close()
	return buf.Bytes(), nil
}
