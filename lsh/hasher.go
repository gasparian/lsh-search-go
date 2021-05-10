package lsh

import (
	"bytes"
	"encoding/gob"
	"errors"
	"gonum.org/v1/gonum/blas/blas64"
	"math"
	"math/rand"
	"sync"
	"time"
)

var (
	dimensionsNumberErr     = errors.New("dimensions number must be a positive integer")
	hasherEmptyInstancesErr = errors.New("hasher must contain at least one instance")
)

// Plane struct holds data needed to work with plane
type plane struct {
	coefs blas64.Vector
	d     float64
}

// HasherInstance holds data for local sensetive hashing algorithm
type hasherInstance struct {
	planes []plane
}

// GetHash calculates LSH code
func (lshInstance *hasherInstance) getHash(inpVec, meanVec blas64.Vector) uint64 {
	shiftedVec := NewVec(make([]float64, inpVec.N))
	blas64.Copy(inpVec, shiftedVec)
	blas64.Axpy(-1.0, meanVec, shiftedVec)
	vec := NewVec(make([]float64, inpVec.N))
	var dp float64
	var dpSign bool
	var hash uint64
	for i, plane := range lshInstance.planes {
		blas64.Copy(shiftedVec, vec) // TODO: do we need this copy?
		dp = blas64.Dot(vec, plane.coefs) - plane.d
		dpSign = math.Signbit(dp)
		if !dpSign {
			hash |= (1 << i)
		}
	}
	return hash
}

type hasherConfig struct {
	NPermutes      int
	NPlanes        int
	BiasMultiplier float64
	Dims           int
}

// Hasher holds N_PERMUTS number of HasherInstance instances
type hasher struct {
	mutex     sync.RWMutex
	config    hasherConfig
	instances []hasherInstance
	bias      float64
	meanVec   blas64.Vector
}

// SafeHashesHolder allows to lock map while write values in it
type safeHashesHolder struct {
	sync.Mutex
	v map[int]uint64
}

// GetRandomPlane generates random coefficients of a plane
func (hasher *hasher) getRandomPlane() blas64.Vector {
	coefs := make([]float64, hasher.config.Dims+1)
	for i := 0; i < hasher.config.Dims; i++ {
		coefs[i] = -1.0 + rand.Float64()*2
	}
	bias := hasher.bias
	coefs[len(coefs)-1] = -1.0*bias + rand.Float64()*bias*2
	return NewVec(coefs)
}

// newHasherInstance creates set of planes which will be used to calculate hash
func (hasher *hasher) newHasherInstance() (hasherInstance, error) {
	if hasher.config.Dims <= 0 {
		return hasherInstance{}, dimensionsNumberErr
	}
	rand.Seed(time.Now().UnixNano())
	lshInstance := hasherInstance{}
	var coefs blas64.Vector
	for i := 0; i < hasher.config.NPlanes; i++ {
		coefs = hasher.getRandomPlane()
		lshInstance.planes = append(lshInstance.planes, plane{
			coefs: NewVec(coefs.Data[:coefs.N-1]),
			d:     coefs.Data[coefs.N-1],
		})
	}
	return lshInstance, nil
}

// Generate method creates the lsh instances
func (hasher *hasher) generate(mean, std []float64) error {
	hasher.mutex.Lock()
	defer hasher.mutex.Unlock()

	convMean := NewVec(mean)
	convStd := NewVec(std)

	hasher.meanVec = convMean
	hasher.bias = blas64.Nrm2(convStd) * hasher.config.BiasMultiplier

	var tmpLsh hasherInstance
	var err error
	for i := 0; i < hasher.config.NPermutes; i++ {
		tmpLsh, err = hasher.newHasherInstance()
		if err != nil {
			return err
		}
		hasher.instances[i] = tmpLsh
	}
	return nil
}

// GetHashes returns map of calculated lsh values for a given vector
func (hasher *hasher) getHashes(vec []float64) map[int]uint64 {
	hasher.mutex.RLock()
	defer hasher.mutex.RUnlock()

	blasVec := NewVec(vec)
	hashes := &safeHashesHolder{v: make(map[int]uint64)}
	wg := sync.WaitGroup{}
	wg.Add(len(hasher.instances))
	for i, hsh := range hasher.instances {
		go func(i int, hsh hasherInstance, hashes *safeHashesHolder) {
			hashes.Lock()
			hashes.v[i] = hsh.getHash(blasVec, hasher.meanVec)
			hashes.Unlock()
			wg.Done()
		}(i, hsh, hashes)
	}
	wg.Wait()
	return hashes.v
}

// Dump encodes Hasher object as a byte-array
func (hasher *hasher) dump() ([]byte, error) {
	hasher.mutex.RLock()
	defer hasher.mutex.RUnlock()

	if len(hasher.instances) == 0 {
		return nil, hasherEmptyInstancesErr
	}
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(hasher)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Load loads Hasher struct from the byte-array file
func (hasher *hasher) load(inp []byte) error {
	hasher.mutex.Lock()
	defer hasher.mutex.Unlock()

	buf := &bytes.Buffer{}
	buf.Write(inp)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&hasher)
	if err != nil {
		return err
	}
	return nil
}
