package lsh

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

	cm "vector-search-go/common"
)

var (
	globMaxNPlanes, _ = strconv.Atoi(os.Getenv("MAX_N_PLANES"))
	globNPermutes, _  = strconv.Atoi(os.Getenv("N_PERMUTS"))
)

// GetPointPlaneDist calculates distance between origin and plane
func GetPointPlaneDist(planeCoefs cm.Vector) cm.Vector {
	values := make([]float64, planeCoefs.Size-1)
	dCoef := planeCoefs.Values[planeCoefs.Size-1]
	var denom float64 = 0.0
	for i := range values {
		denom += planeCoefs.Values[i] * planeCoefs.Values[i]
	}
	for i := range values {
		values[i] = planeCoefs.Values[i] * dCoef / denom
	}
	return cm.Vector{
		Values: values,
		Size:   len(values),
	}
}

// NewLSHIndexInstance creates new instance of LSHIndex object
func NewLSHIndexInstance(meanVec, stdVec cm.Vector, maxNPlanes int) (LSHIndexInstance, error) {
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
func NewLSHIndex(convMean, convStd cm.Vector) (*LSHIndex, error) {
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

func (lsh *LSHIndexInstance) getRandomPlane() cm.Vector {
	coefs := cm.Vector{
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
	var coefs cm.Vector
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
func (lsh *LSHIndexInstance) GetHash(inpVec *cm.Vector) uint64 {
	var hash uint64
	var vec cm.Vector
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
