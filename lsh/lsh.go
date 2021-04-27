package lsh

import (
	"bytes"
	"encoding/gob"
	"errors"
	cmap "github.com/orcaman/concurrent-map"
	"gonum.org/v1/gonum/blas/blas64"
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	Cosine = iota
	Euclidian
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

// Config holds all needed constants for creating the Hasher instance
type Config struct {
	DistanceMetric int
	NPermutes      int
	NPlanes        int
	BiasMultiplier float64
	DistanceThrsh  float64
	Dims           int
}

// Hasher holds N_PERMUTS number of HasherInstance instances
type Hasher struct {
	mutex     sync.RWMutex
	Config    Config
	Instances []HasherInstance
	Bias      float64
	MeanVec   blas64.Vector
}

// LSHIndex holds buckets with vectors and hasher instance
// TODO:
type LSHIndex struct {
	Index  cmap.ConcurrentMap
	Hasher Hasher
}

// SafeHashesHolder allows to lock map while write values in it
type safeHashesHolder struct {
	sync.Mutex
	v map[int]uint64
}

// GetHash calculates LSH code
func (lshInstance *HasherInstance) GetHash(inpVec, meanVec blas64.Vector) uint64 {
	var hash uint64
	shiftedVec := NewVec(make([]float64, inpVec.N))
	blas64.Copy(inpVec, shiftedVec)
	blas64.Axpy(-1.0, meanVec, shiftedVec)
	vec := NewVec(make([]float64, inpVec.N))
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

// New creates slice of hasher instances to hold several permutations results
func New(config Config) *Hasher {
	hasher := &Hasher{
		Config:    config,
		Instances: make([]HasherInstance, config.NPermutes),
	}
	return hasher
}

// GetRandomPlane generates random coefficients of a plane
func (hasher *Hasher) getRandomPlane() blas64.Vector {
	coefs := make([]float64, hasher.Config.Dims+1)
	for i := 0; i < hasher.Config.Dims; i++ {
		coefs[i] = -1.0 + rand.Float64()*2
	}
	bias := hasher.Bias
	coefs[len(coefs)-1] = -1.0*bias + rand.Float64()*bias*2
	return NewVec(coefs)
}

// newHasherInstance creates set of planes which will be used to calculate hash
func (hasher *Hasher) newHasherInstance() (HasherInstance, error) {
	if hasher.Config.Dims <= 0 {
		return HasherInstance{}, errors.New("dimensions number must be a positive integer")
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
func (hasher *Hasher) Generate(mean, std []float64) error {
	hasher.mutex.Lock()
	defer hasher.mutex.Unlock()

	convMean := NewVec(mean)
	convStd := NewVec(std)

	if hasher.Config.DistanceMetric == Cosine {
		blas64.Scal(0.0, convStd)
	}
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
func (hasher *Hasher) GetHashes(vec []float64) map[int]uint64 {
	hasher.mutex.RLock()
	defer hasher.mutex.RUnlock()

	blasVec := NewVec(vec)
	hashes := &safeHashesHolder{v: make(map[int]uint64)}
	wg := sync.WaitGroup{}
	wg.Add(len(hasher.Instances))
	for i, lshInstance := range hasher.Instances {
		go func(i int, lsh HasherInstance, hashes *safeHashesHolder) {
			hashes.Lock()
			hashes.v[i] = lsh.GetHash(blasVec, hasher.MeanVec)
			hashes.Unlock()
			wg.Done()
		}(i, lshInstance, hashes)
	}
	wg.Wait()
	return hashes.v
}

// GetDist measures the distance by specified distance metric
func (hasher *Hasher) GetDist(lv, rv []float64) (float64, bool) {
	hasher.mutex.Lock()
	defer hasher.mutex.Unlock()

	lvBlas := NewVec(lv)
	rvBlas := NewVec(rv)

	var dist float64 = 0.0
	switch hasher.Config.DistanceMetric {
	case Cosine:
		if IsZeroVectorBlas(lvBlas) || IsZeroVectorBlas(rvBlas) {
			return 1.0, false // NOTE: zero vectors are wrong with angular metric
		}
		dist = CosineSim(lvBlas, rvBlas)
	case Euclidian:
		dist = L2(lvBlas, rvBlas)
	}
	if dist <= hasher.Config.DistanceThrsh {
		return dist, true
	}
	return dist, false
}

// Dump encodes Hasher object as a byte-array
func (hasher *Hasher) Dump() ([]byte, error) {
	hasher.mutex.RLock()
	defer hasher.mutex.RUnlock()

	if len(hasher.Instances) == 0 {
		return nil, errors.New("hasher must contain at least one instance")
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
func (hasher *Hasher) Load(inp []byte) error {
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
