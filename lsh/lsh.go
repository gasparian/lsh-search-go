package lsh

import (
	"bytes"
	"encoding/gob"
	"errors"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"gonum.org/v1/gonum/blas/blas64"
	cm "lsh-search-service/common"
)

// GetHash calculates LSH code
func (lshInstance *HasherInstance) GetHash(inpVec, meanVec blas64.Vector) uint64 {
	var hash uint64
	shiftedVec := cm.NewVec(make([]float64, inpVec.N))
	blas64.Copy(inpVec, shiftedVec)
	blas64.Axpy(-1.0, meanVec, shiftedVec)
	vec := cm.NewVec(make([]float64, inpVec.N))
	var dp float64
	var dpSign bool
	for i, plane := range lshInstance.Planes {
		blas64.Copy(shiftedVec, vec)
		dp = blas64.Dot(vec, plane.Coefs) - plane.D
		dpSign = math.Signbit(dp)
		if !dpSign {
			hash |= (1 << i)
		}
	}
	return hash
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

// GetRandomPlane generates random coefficients of a plane
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

// newHasherInstance creates set of planes which will be used to calculate hash
func (lshIndex *Hasher) newHasherInstance() (HasherInstance, error) {
	if lshIndex.Config.Dims <= 0 {
		return HasherInstance{}, errors.New("dimensions number must be a positive integer")
	}
	rand.Seed(time.Now().UnixNano())
	lshInstance := HasherInstance{}
	var coefs blas64.Vector
	for i := 0; i < lshIndex.Config.NPlanes; i++ {
		coefs = lshIndex.getRandomPlane()
		lshInstance.Planes = append(lshInstance.Planes, Plane{
			Coefs: cm.NewVec(coefs.Data[:coefs.N-1]),
			D:     coefs.Data[coefs.N-1],
		})
	}
	return lshInstance, nil
}

// Generate method creates the lsh instances
func (lshIndex *Hasher) Generate(convMean, convStd blas64.Vector) error {
	lshIndex.Lock()
	defer lshIndex.Unlock()

	if lshIndex.Config.IsAngularDistance == 1 {
		blas64.Scal(0.0, convStd)
	}
	lshIndex.Config.MeanVec = convMean
	lshIndex.Config.Bias = blas64.Nrm2(convStd) * lshIndex.Config.BiasMultiplier

	var tmpLSHIndex HasherInstance
	var err error
	for i := 0; i < lshIndex.Config.NPermutes; i++ {
		tmpLSHIndex, err = lshIndex.newHasherInstance()
		if err != nil {
			return err
		}
		lshIndex.Instances[i] = tmpLSHIndex
		lshIndex.HashFieldsNames[i] = strconv.Itoa(i)
	}
	return nil
}

// GetHashes returns map of calculated lsh values
func (lshIndex *Hasher) GetHashes(vec blas64.Vector) map[int]uint64 {
	lshIndex.Lock()
	defer lshIndex.Unlock()

	hashes := safeHashesHolder{v: make(map[int]uint64)}
	var wg sync.WaitGroup
	for i, lshInstance := range lshIndex.Instances {
		wg.Add(1)
		go func(idx int, lsh *HasherInstance, hashesMap *safeHashesHolder) {
			hashesMap.Lock()
			hashesMap.v[idx] = lsh.GetHash(vec, lshIndex.Config.MeanVec)
			hashesMap.Unlock()
			wg.Done()
		}(i, &lshInstance, &hashes)
	}
	wg.Wait()
	return hashes.v
}

// GetDist returns measure of the specified distance metric
func (lshIndex *Hasher) GetDist(lv, rv blas64.Vector) (float64, bool) {
	lshIndex.Lock()
	defer lshIndex.Unlock()
	var dist float64 = 0.0
	if lshIndex.Config.IsAngularDistance == 1 {
		if cm.IsZeroVector(lv) || cm.IsZeroVector(rv) {
			return 1.0, false // NOTE: zero vectors are wrong with angular metric
		}
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
	// TODO: maybe it's possible to get rid of helper-struct and make mutex a field
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
