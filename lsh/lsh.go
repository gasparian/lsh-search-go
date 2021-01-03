package lsh

import (
	"bytes"
	"encoding/gob"
	"errors"
	"math"
	"math/rand"
	"strconv"
	"time"

	"gonum.org/v1/gonum/blas/blas64"
	cm "lsh-search-service/common"
)

// getPointPlaneDist calculates distance between origin and plane
func getPointPlaneDist(planeCoefs blas64.Vector) blas64.Vector {
	values := make([]float64, planeCoefs.N-1)
	dCoef := planeCoefs.Data[planeCoefs.N-1]
	var denom float64 = 0.0
	for i := range values {
		denom += planeCoefs.Data[i] * planeCoefs.Data[i]
	}
	for i := range values {
		values[i] = planeCoefs.Data[i] * dCoef / denom
	}
	return cm.NewVec(values)
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

func (lshIndex *Hasher) getRandomPlane() blas64.Vector {
	coefs := make([]float64, lshIndex.Config.Dims+1)
	var l2 float64 = 0.0
	for i := 0; i < lshIndex.Config.Dims; i++ {
		coefs[i] = -1.0 + rand.Float64()*2
		l2 += coefs[i] * coefs[i]
	}
	l2 = math.Sqrt(l2)
	bias := l2 * lshIndex.Config.Bias
	coefs[len(coefs)-1] = -1.0*bias + rand.Float64()*bias*2
	return cm.NewVec(coefs)
}

// NewHasherInstance creates set of planes which will be used to calculate hash
func (lshIndex *Hasher) NewHasherInstance() (HasherInstance, error) {
	if lshIndex.Config.Dims <= 0 {
		return HasherInstance{}, errors.New("dimensions number must be a positive integer")
	}
	rand.Seed(time.Now().UnixNano())
	lshInstance := HasherInstance{}
	var coefs blas64.Vector
	for i := 0; i < lshIndex.Config.NPlanes; i++ {
		coefs = lshIndex.getRandomPlane()
		lshInstance.Planes = append(lshInstance.Planes, Plane{
			Coefs:      coefs,
			InnerPoint: getPointPlaneDist(coefs),
		})
	}
	return lshInstance, nil
}

// getHash calculates LSH code
func (lshInstance *HasherInstance) getHash(inpVec, meanVec blas64.Vector) uint64 {
	var hash uint64
	vec := cm.NewVec(make([]float64, inpVec.N))
	var dpSign bool
	for i, plane := range lshInstance.Planes {
		blas64.Copy(inpVec, vec)
		blas64.Axpy(-1.0, meanVec, vec)
		blas64.Axpy(-1.0, plane.InnerPoint, vec)
		dpSign = math.Signbit(blas64.Dot(vec, plane.Coefs))
		if !dpSign {
			hash |= (1 << i)
		}
	}
	return hash
}

// Generate method creates the lsh instances
func (lshIndex *Hasher) Generate(convMean, convStd blas64.Vector) error {
	lshIndex.Lock()
	defer lshIndex.Unlock()

	if lshIndex.Config.IsAngularDistance == 1 {
		blas64.Scal(0.0, convStd)
	}
	lshIndex.Config.MeanVec = convMean
	lshIndex.Config.Bias = blas64.Nrm2(convStd) * float64(lshIndex.Config.BiasMultiplier) // TO DO: why l2 norm is here? maybe mean or max of this vector?

	var tmpLSHIndex HasherInstance
	var err error
	for i := 0; i < lshIndex.Config.NPermutes; i++ {
		tmpLSHIndex, err = lshIndex.NewHasherInstance()
		if err != nil {
			return err
		}
		lshIndex.Instances[i] = tmpLSHIndex
		lshIndex.HashFieldsNames[i] = strconv.Itoa(i)
	}
	return nil
}

// GetHashes returns map of calculated lsh values
func (lshIndex *Hasher) GetHashes(vec blas64.Vector) (map[int]uint64, error) {
	lshIndex.Lock()
	defer lshIndex.Unlock()

	var result map[int]uint64
	for idx, lshInstance := range lshIndex.Instances {
		result[idx] = lshInstance.getHash(vec, lshIndex.Config.MeanVec)
	}
	return result, nil
}

// GetDist returns measure of the specified distance metric
func (lshIndex *Hasher) GetDist(lv, rv blas64.Vector) (float64, bool) {
	lshIndex.Lock()
	defer lshIndex.Unlock()
	var dist float64 = 0.0
	if lshIndex.Config.IsAngularDistance == 1 {
		dist = cm.CosineSim(lv, rv)
	} else {
		dist = cm.L2(lv, rv)
	}
	if dist <= lshIndex.Config.DistanceThrsh {
		return dist, true
	}
	return dist, false
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
		Instances:       &lshIndex.Instances,
		HashFieldsNames: &lshIndex.HashFieldsNames,
		Config:          &lshIndex.Config,
	}
	err := enc.Encode(encodable)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Load loads Hasher struct from the byte-array file
func (lshIndex *Hasher) Load(inp []byte) error {
	lshIndex.Lock()
	defer lshIndex.Unlock()

	buf := &bytes.Buffer{}
	buf.Write(inp)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&lshIndex)
	if err != nil {
		return err
	}
	return nil
}
