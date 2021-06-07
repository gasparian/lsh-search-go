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
	N blas64.Vector
	D float64
}

// HasherInstance holds data for local sensetive hashing algorithm
type HasherInstance struct {
	Planes []Plane
}

// GetHash calculates LSH code
func (lshInstance *HasherInstance) getHash(inpVec blas64.Vector) uint64 {
	vec := NewVec(make([]float64, inpVec.N))
	var dp float64
	var dpSign bool
	var hash uint64
	for i, plane := range lshInstance.Planes {
		blas64.Copy(inpVec, vec)
		dp = blas64.Dot(vec, plane.N) + plane.D
		dpSign = math.Signbit(dp) // NOTE: returns true if dp < 0
		if dpSign {
			hash |= (1 << i)
		}
	}
	return hash
}

type HasherConfig struct {
	NPermutes                 int
	NPlanes                   int
	Dims                      int
	PlaneOriginDistMultiplier float64
}

// Hasher holds N_PERMUTS number of HasherInstance instances
type Hasher struct {
	mutex     sync.RWMutex
	Config    HasherConfig
	Instances []HasherInstance
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
func (hasher *Hasher) getRandomPlane() Plane {
	plane := Plane{N: NewVec(make([]float64, hasher.Config.Dims))}
	for i := 0; i < hasher.Config.Dims; i++ {
		plane.N.Data[i] = -1.0 + rand.Float64()*2
	}
	maxD := hasher.Config.PlaneOriginDistMultiplier * blas64.Nrm2(plane.N)
	plane.D = -1.0 * (-1.0*maxD + rand.Float64()*maxD*2)
	return plane
}

// newHasherInstance creates set of planes which will be used to calculate hash
func (hasher *Hasher) newHasherInstance() (HasherInstance, error) {
	if hasher.Config.Dims <= 0 {
		return HasherInstance{}, dimensionsNumberErr
	}
	rand.Seed(time.Now().UnixNano())
	lshInstance := HasherInstance{}
	for i := 0; i < hasher.Config.NPlanes; i++ {
		plane := hasher.getRandomPlane()
		lshInstance.Planes = append(lshInstance.Planes, plane)
	}
	return lshInstance, nil
}

// Generate method creates the hasher instances
func (hasher *Hasher) generate() error {
	hasher.mutex.Lock()
	defer hasher.mutex.Unlock()

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
func (hasher *Hasher) getHashes(vec blas64.Vector) map[int]uint64 {
	hasher.mutex.RLock()
	defer hasher.mutex.RUnlock()

	hashes := &safeHashesHolder{v: make(map[int]uint64)}
	wg := sync.WaitGroup{}
	wg.Add(len(hasher.Instances))
	for i, hsh := range hasher.Instances {
		go func(i int, hsh HasherInstance, hashes *safeHashesHolder) {
			defer wg.Done()
			hashes.Lock()
			hashes.v[i] = hsh.getHash(vec)
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
