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

	cm "lsh-search-engine/common"
)

// GetPointPlaneDist calculates distance between origin and plane
func GetPointPlaneDist(planeCoefs cm.Vector) cm.Vector {
	values := make(cm.Vector, len(planeCoefs)-1)
	dCoef := planeCoefs[len(planeCoefs)-1]
	var denom float64 = 0.0
	for i := range values {
		denom += planeCoefs[i] * planeCoefs[i]
	}
	for i := range values {
		values[i] = planeCoefs[i] * dCoef / denom
	}
	return values
}

// NewLSHIndexInstance creates new instance of Hasher object
func NewLSHIndexInstance(meanVec, stdVec cm.Vector, maxNPlanes int) (HasherInstance, error) {
	lshIndex := HasherInstance{
		Dims:    len(meanVec),
		Bias:    stdVec.L2Norm(),
		MeanVec: meanVec,
		NPlanes: maxNPlanes,
	}
	err := lshIndex.Build()
	if err != nil {
		return HasherInstance{}, err
	}
	return lshIndex, nil
}

// NewLSHIndex creates slice of LSHIndexInstances to hold several permutations results
func NewLSHIndex(config Config) *Hasher {
	lshIndex := &Hasher{
		Config:          config,
		Instances:       make([]HasherInstance, config.NPermutes),
		HashFieldsNames: make([]string, config.NPermutes),
	}
	return lshIndex
}

func (lsh *HasherInstance) getRandomPlane() cm.Vector {
	coefs := make(cm.Vector, lsh.Dims+1)
	var l2 float64 = 0.0
	for i := 0; i < lsh.Dims; i++ {
		coefs[i] = -1.0 + rand.Float64()*2
		l2 += coefs[i] * coefs[i]
	}
	l2 = math.Sqrt(l2)
	bias := l2 * lsh.Bias
	coefs[len(coefs)-1] = -1.0*bias + rand.Float64()*bias*2
	return coefs
}

// Build creates set of planes which will be used to calculate hash
func (lsh *HasherInstance) Build() error {
	if lsh.Dims <= 0 {
		return errors.New("dimensions number must be a positive integer")
	}

	rand.Seed(time.Now().UnixNano())
	var coefs cm.Vector
	for i := 0; i < lsh.NPlanes; i++ {
		coefs = lsh.getRandomPlane()
		lsh.Planes = append(lsh.Planes, Plane{
			Coefs:      coefs,
			InnerPoint: GetPointPlaneDist(coefs),
		})
	}
	return nil
}

// GetHash calculates LSH code
func (lsh *HasherInstance) GetHash(inpVec cm.Vector) uint64 {
	var hash uint64
	var vec cm.Vector
	var plane *Plane
	var dpSign bool
	for i := 0; i < lsh.NPlanes; i++ {
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

// Generate method creates the lsh instances
func (lshIndex *Hasher) Generate(convMean, convStd cm.Vector) error {
	if lshIndex.Config.IsAngularDistance == 1 {
		convStd = convStd.ConstMul(0.0)
	}
	var tmpLSHIndex HasherInstance
	var err error
	for i := 0; i < lshIndex.Config.NPermutes; i++ {
		tmpLSHIndex, err = NewLSHIndexInstance(convMean, convStd, lshIndex.Config.MaxNPlanes)
		if err != nil {
			return err
		}
		lshIndex.Instances[i] = tmpLSHIndex
		lshIndex.HashFieldsNames[i] = strconv.Itoa(i)
	}
	return nil
}

// GetHashes returns map of calculated lsh values
func (lshIndex *Hasher) GetHashes(vec cm.Vector) (map[int]uint64, error) {
	var result map[int]uint64
	for idx, lshInstance := range lshIndex.Instances {
		result[idx] = lshInstance.GetHash(vec)
	}
	return result, nil
}

// GetDist returns measure of the specified distance metric
func (lshIndex *Hasher) GetDist(lv, rv cm.Vector) float64 {
	if lshIndex.Config.IsAngularDistance == 1 {
		return lv.CosineSim(rv)
	}
	return lv.L2(rv)
}

// Dump encodes Hasher object as a byte-array
func (lshIndex *Hasher) Dump() ([]byte, error) {
	lshIndex.Lock()
	defer lshIndex.Unlock()

	if len(lshIndex.Instances) == 0 {
		return nil, errors.New("search index must contain at least one object")
	}
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	encodable := HasherEncode{
		Instances: &lshIndex.Instances,
	}
	err := enc.Encode(encodable)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Load loads Hasher struct from the byte-array file
func (lshIndex *Hasher) Load(inp []byte) error {
	buf := &bytes.Buffer{}
	buf.Write(inp)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&lshIndex)
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
