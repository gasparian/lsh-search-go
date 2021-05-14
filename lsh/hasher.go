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
type Plane struct {
	Coefs blas64.Vector
	D     float64
}

// HasherInstance holds data for local sensetive hashing algorithm
type HasherInstance struct {
	Planes []Plane
}

// GetHash calculates LSH code
func (lshInstance *HasherInstance) getHash(inpVec, meanVec blas64.Vector) uint64 {
	shiftedVec := NewVec(make([]float64, inpVec.N))
	blas64.Copy(inpVec, shiftedVec)
	blas64.Axpy(-1.0, meanVec, shiftedVec)
	vec := NewVec(make([]float64, inpVec.N))
	var dp float64
	var dpSign bool
	var hash uint64
	for i, plane := range lshInstance.Planes {
		blas64.Copy(shiftedVec, vec) // TODO: do we really need this copy?
		dp = blas64.Dot(vec, plane.Coefs) - plane.D
		dpSign = math.Signbit(dp)
		if !dpSign {
			hash |= (1 << i)
		}
	}
	return hash
}

type HasherConfig struct {
	NPermutes      int
	NPlanes        int
	BiasMultiplier float64
	Dims           int
}

// Hasher holds N_PERMUTS number of HasherInstance instances
type Hasher struct {
	mutex     sync.RWMutex
	Config    HasherConfig
	Instances []HasherInstance
	Bias      float64
	MeanVec   blas64.Vector
}

func NewHasher(config HasherConfig) *Hasher {
	return &Hasher{
		Config:    config,
		Instances: make([]HasherInstance, config.NPermutes),
	}
}

// SafeHashesHolder allows to lock map while write values in it
type safeHashesHolder struct {
	sync.Mutex
	v map[int]uint64
}

// GetRandomPlane generates random coefficients of a plane
func (hasher *Hasher) getRandomPlane() blas64.Vector {
	coefs := make([]float64, hasher.Config.Dims+1)
	for i := 0; i < hasher.Config.Dims; i++ {
		coefs[i] = -1.0 + rand.Float64()*2
	}
	coefs[len(coefs)-1] = -1.0*hasher.Bias + rand.Float64()*hasher.Bias*2
	return NewVec(coefs)
}

// newHasherInstance creates set of planes which will be used to calculate hash
func (hasher *Hasher) newHasherInstance() (HasherInstance, error) {
	if hasher.Config.Dims <= 0 {
		return HasherInstance{}, dimensionsNumberErr
	}
	rand.Seed(time.Now().UnixNano())
	lshInstance := HasherInstance{}
	var coefs blas64.Vector
	for i := 0; i < hasher.Config.NPlanes; i++ {
		coefs = hasher.getRandomPlane()
		lshInstance.Planes = append(lshInstance.Planes, Plane{
			Coefs: NewVec(coefs.Data[:coefs.N-1]),
			D:     coefs.Data[coefs.N-1],
		})
	}
	return lshInstance, nil
}

// Generate method creates the lsh instances
func (hasher *Hasher) generate(mean, std []float64) error {
	hasher.mutex.Lock()
	defer hasher.mutex.Unlock()

	convMean := NewVec(mean)
	convStd := NewVec(std)

	hasher.MeanVec = convMean
	hasher.Bias = blas64.Nrm2(convStd) * hasher.Config.BiasMultiplier

	var tmpLsh HasherInstance
	var err error
	for i := 0; i < hasher.Config.NPermutes; i++ {
		tmpLsh, err = hasher.newHasherInstance()
		if err != nil {
			return err
		}
		hasher.Instances[i] = tmpLsh
	}
	return nil
}

// GetHashes returns map of calculated lsh values for a given vector
func (hasher *Hasher) getHashes(vec []float64) map[int]uint64 {
	hasher.mutex.RLock()
	defer hasher.mutex.RUnlock()

	blasVec := NewVec(vec)
	hashes := &safeHashesHolder{v: make(map[int]uint64)}
	wg := sync.WaitGroup{}
	wg.Add(len(hasher.Instances))
	for i, hsh := range hasher.Instances {
		go func(i int, hsh HasherInstance, hashes *safeHashesHolder) {
			defer wg.Done()
			hashes.Lock()
			hashes.v[i] = hsh.getHash(blasVec, hasher.MeanVec)
			hashes.Unlock()
		}(i, hsh, hashes)
	}
	wg.Wait()
	return hashes.v
}

// Dump encodes Hasher object as a byte-array
func (hasher *Hasher) dump() ([]byte, error) {
	hasher.mutex.RLock()
	defer hasher.mutex.RUnlock()

	if len(hasher.Instances) == 0 {
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
func (hasher *Hasher) load(inp []byte) error {
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
